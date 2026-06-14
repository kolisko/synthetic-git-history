import tempfile
import unittest
from pathlib import Path

from synthgit.config import ConfigError, load_config


class ConfigTests(unittest.TestCase):
    def test_loads_example_config(self):
        config = load_config(Path("config.example.toml"))

        self.assertEqual(config.repository.branch, "main")
        self.assertEqual(config.date_range.start.isoformat(), "2010-01-01")
        self.assertGreaterEqual(config.volume.max_commits_per_day, config.volume.min_commits_per_day)

    def test_rejects_absolute_activity_file(self):
        with tempfile.TemporaryDirectory() as tmp:
            config_path = Path(tmp) / "config.toml"
            config_path.write_text(
                """
[repository]
path = "./repo"

[identity]
name = "Bot"
email = "bot@example.invalid"

[range]
start = "2020-01-01"
end = "2020-01-01"

[content]
activity_file = "/tmp/activity.log"
message_templates = ["test"]
""",
                encoding="utf-8",
            )

            with self.assertRaises(ConfigError):
                load_config(config_path)


if __name__ == "__main__":
    unittest.main()

