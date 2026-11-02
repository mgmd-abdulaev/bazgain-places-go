# Local database baseline

These SQL files initialize only a new disposable Docker Compose database. They are a compatibility baseline for developing and testing the five-endpoint migration candidate.

Do not run them against staging, production, or any database with existing data. A real deployment must use the owning project's reviewed migration history and backup/rollback process.

`003_seed.sql` contains synthetic development fixtures and no production export.

