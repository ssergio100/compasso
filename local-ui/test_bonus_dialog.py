import unittest

from bonus_dialog import BONUS_OPTIONS, seconds_for_index


class BonusDialogLogicTest(unittest.TestCase):
    def test_all_options_convert_to_exact_seconds(self):
        self.assertEqual(
            [seconds_for_index(index) for index in range(len(BONUS_OPTIONS))],
            [900, 1800, 3600, 7200],
        )

    def test_invalid_option_is_rejected(self):
        with self.assertRaises(ValueError):
            seconds_for_index(-1)


if __name__ == "__main__":
    unittest.main()
