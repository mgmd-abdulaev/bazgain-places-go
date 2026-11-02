package postgres

import (
	"context"
	"fmt"

	"github.com/tx-mi/bazgain-places-go/internal/places"
)

func (r *Repository) loadListRelations(ctx context.Context, items []places.PlaceRecord) error {
	if len(items) == 0 {
		return nil
	}
	loaders := []struct {
		name string
		fn   func(context.Context, []places.PlaceRecord) error
	}{{"badges", r.loadBadges}, {"hashtags", r.loadHashtags}, {"memberships", func(ctx context.Context, items []places.PlaceRecord) error {
		return r.loadMemberships(ctx, items, true)
	}}, {"opening periods", r.loadOpeningPeriods}}
	for _, loader := range loaders {
		if err := loader.fn(ctx, items); err != nil {
			return wrapLoad(loader.name, err)
		}
	}
	return nil
}

func (r *Repository) loadAllRelations(ctx context.Context, items []places.PlaceRecord, publishedCollectionsOnly bool) error {
	if err := r.loadListRelationsWithMembershipMode(ctx, items, publishedCollectionsOnly); err != nil {
		return err
	}
	loaders := []struct {
		name string
		fn   func(context.Context, []places.PlaceRecord) error
	}{{"gallery", r.loadGallery}, {"contacts", r.loadContacts}, {"info blocks", r.loadInfoBlocks}}
	for _, loader := range loaders {
		if err := loader.fn(ctx, items); err != nil {
			return wrapLoad(loader.name, err)
		}
	}
	return nil
}

func (r *Repository) loadListRelationsWithMembershipMode(ctx context.Context, items []places.PlaceRecord, publishedOnly bool) error {
	if len(items) == 0 {
		return nil
	}
	if err := r.loadBadges(ctx, items); err != nil {
		return wrapLoad("badges", err)
	}
	if err := r.loadHashtags(ctx, items); err != nil {
		return wrapLoad("hashtags", err)
	}
	if err := r.loadMemberships(ctx, items, publishedOnly); err != nil {
		return wrapLoad("memberships", err)
	}
	if err := r.loadOpeningPeriods(ctx, items); err != nil {
		return wrapLoad("opening periods", err)
	}
	return nil
}

func (r *Repository) loadGallery(ctx context.Context, items []places.PlaceRecord) error {
	rows, err := r.pool.Query(ctx, `SELECT id::text,place_id::text,kind::text,url,thumbnail_url,width,height,alt_text,sort_order FROM place_media WHERE place_id=ANY($1::uuid[]) ORDER BY place_id,sort_order,id`, idsOf(items))
	if err != nil {
		return err
	}
	defer rows.Close()
	index := indexOf(items)
	for rows.Next() {
		var row places.MediaRecord
		var placeID string
		if err := rows.Scan(&row.ID, &placeID, &row.Kind, &row.URL, &row.ThumbnailURL, &row.Width, &row.Height, &row.AltText, &row.SortOrder); err != nil {
			return err
		}
		index[placeID].Gallery = append(index[placeID].Gallery, row)
	}
	return rows.Err()
}
func (r *Repository) loadBadges(ctx context.Context, items []places.PlaceRecord) error {
	rows, err := r.pool.Query(ctx, `SELECT id::text,place_id::text,title,tone::text,sort_order FROM place_badges WHERE place_id=ANY($1::uuid[]) ORDER BY place_id,sort_order,id`, idsOf(items))
	if err != nil {
		return err
	}
	defer rows.Close()
	index := indexOf(items)
	for rows.Next() {
		var row places.BadgeRecord
		var placeID string
		if err := rows.Scan(&row.ID, &placeID, &row.Title, &row.Tone, &row.SortOrder); err != nil {
			return err
		}
		index[placeID].Badges = append(index[placeID].Badges, row)
	}
	return rows.Err()
}
func (r *Repository) loadContacts(ctx context.Context, items []places.PlaceRecord) error {
	rows, err := r.pool.Query(ctx, `SELECT id::text,place_id::text,type::text,label,display_value,action_url,sort_order FROM place_contacts WHERE place_id=ANY($1::uuid[]) ORDER BY place_id,sort_order,id`, idsOf(items))
	if err != nil {
		return err
	}
	defer rows.Close()
	index := indexOf(items)
	for rows.Next() {
		var row places.ContactRecord
		var placeID string
		if err := rows.Scan(&row.ID, &placeID, &row.Type, &row.Label, &row.DisplayValue, &row.ActionURL, &row.SortOrder); err != nil {
			return err
		}
		index[placeID].Contacts = append(index[placeID].Contacts, row)
	}
	return rows.Err()
}
func (r *Repository) loadHashtags(ctx context.Context, items []places.PlaceRecord) error {
	rows, err := r.pool.Query(ctx, `SELECT t.id::text,a.place_id::text,t.slug,t.title,a.sort_order FROM place_tag_assignments a JOIN place_tags t ON t.id=a.tag_id WHERE a.place_id=ANY($1::uuid[]) ORDER BY a.place_id,a.sort_order,t.id`, idsOf(items))
	if err != nil {
		return err
	}
	defer rows.Close()
	index := indexOf(items)
	for rows.Next() {
		var row places.HashtagRecord
		var placeID string
		if err := rows.Scan(&row.ID, &placeID, &row.Slug, &row.Title, &row.SortOrder); err != nil {
			return err
		}
		index[placeID].Hashtags = append(index[placeID].Hashtags, row)
	}
	return rows.Err()
}
func (r *Repository) loadInfoBlocks(ctx context.Context, items []places.PlaceRecord) error {
	rows, err := r.pool.Query(ctx, `SELECT b.id::text,a.place_id::text,b.type::text,b.icon_key,b.title,b.value_markdown,b.background_color_light,b.background_color_dark,a.sort_order FROM place_info_block_assignments a JOIN info_blocks b ON b.id=a.info_block_id WHERE a.place_id=ANY($1::uuid[]) ORDER BY a.place_id,a.sort_order,b.id`, idsOf(items))
	if err != nil {
		return err
	}
	defer rows.Close()
	index := indexOf(items)
	for rows.Next() {
		var row places.InfoBlockRecord
		var placeID string
		if err := rows.Scan(&row.ID, &placeID, &row.Type, &row.IconKey, &row.Title, &row.ValueMarkdown, &row.BackgroundColorLight, &row.BackgroundColorDark, &row.SortOrder); err != nil {
			return err
		}
		index[placeID].InfoBlocks = append(index[placeID].InfoBlocks, row)
	}
	return rows.Err()
}

func (r *Repository) loadMemberships(ctx context.Context, items []places.PlaceRecord, publishedOnly bool) error {
	condition := ""
	if publishedOnly {
		condition = " AND c.status='published'::place_collection_status"
	}
	query := fmt.Sprintf(`SELECT m.id::text,m.place_id::text,m.collection_id::text,m.sort_order FROM place_collection_memberships m JOIN place_collections c ON c.id=m.collection_id WHERE m.place_id=ANY($1::uuid[])%s ORDER BY m.place_id,m.sort_order,m.id`, condition)
	rows, err := r.pool.Query(ctx, query, idsOf(items))
	if err != nil {
		return err
	}
	defer rows.Close()
	index := indexOf(items)
	type membershipLocation struct {
		placeID string
		index   int
	}
	membershipIndex := make(map[string]membershipLocation)
	membershipIDs := make([]string, 0)
	for rows.Next() {
		var row places.MembershipRecord
		var placeID string
		if err := rows.Scan(&row.ID, &placeID, &row.CollectionID, &row.SortOrder); err != nil {
			return err
		}
		target := index[placeID]
		target.Memberships = append(target.Memberships, row)
		membershipIndex[row.ID] = membershipLocation{placeID: placeID, index: len(target.Memberships) - 1}
		membershipIDs = append(membershipIDs, row.ID)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if len(membershipIDs) == 0 {
		return nil
	}
	filters, err := r.pool.Query(ctx, `SELECT id::text,membership_id::text,filter_id::text,sort_order FROM place_collection_filter_memberships WHERE membership_id=ANY($1::uuid[]) ORDER BY membership_id,sort_order,id`, membershipIDs)
	if err != nil {
		return err
	}
	defer filters.Close()
	for filters.Next() {
		var row places.FilterMembershipRecord
		var membershipID string
		if err := filters.Scan(&row.ID, &membershipID, &row.FilterID, &row.SortOrder); err != nil {
			return err
		}
		if location, ok := membershipIndex[membershipID]; ok {
			parent := index[location.placeID]
			parent.Memberships[location.index].FilterMemberships = append(parent.Memberships[location.index].FilterMemberships, row)
		}
	}
	return filters.Err()
}

func (r *Repository) loadOpeningPeriods(ctx context.Context, items []places.PlaceRecord) error {
	index := indexOf(items)
	rows, err := r.pool.Query(ctx, `SELECT id::text,place_id::text,open_day::text,open_minutes,close_day::text,close_minutes,sort_order FROM place_opening_periods WHERE place_id=ANY($1::uuid[]) ORDER BY place_id,sort_order,id`, idsOf(items))
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var row places.OpeningPeriodRecord
		var placeID string
		if err := rows.Scan(&row.ID, &placeID, &row.OpenDay, &row.OpenMinutes, &row.CloseDay, &row.CloseMinutes, &row.SortOrder); err != nil {
			return err
		}
		index[placeID].RegularPeriods = append(index[placeID].RegularPeriods, row)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	special, err := r.pool.Query(ctx, `SELECT id::text,place_id::text,start_date,end_date,closed,open_minutes,close_minutes,note_markdown,sort_order FROM place_special_opening_periods WHERE place_id=ANY($1::uuid[]) ORDER BY place_id,sort_order,id`, idsOf(items))
	if err != nil {
		return err
	}
	defer special.Close()
	for special.Next() {
		var row places.SpecialOpeningPeriodRecord
		var placeID string
		if err := special.Scan(&row.ID, &placeID, &row.StartDate, &row.EndDate, &row.Closed, &row.OpenMinutes, &row.CloseMinutes, &row.NoteMarkdown, &row.SortOrder); err != nil {
			return err
		}
		index[placeID].SpecialPeriods = append(index[placeID].SpecialPeriods, row)
	}
	return special.Err()
}
