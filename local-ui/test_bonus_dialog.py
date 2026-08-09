import unittest
from unittest import mock

from bonus_dialog import (
    BONUS_OPTIONS,
    SETUP_APPLICATION_PATH,
    open_settings,
    friendly_error_name,
    seconds_for_index,
    unavailable_service_message,
)


class BonusDialogLogicTest(unittest.TestCase):
    def test_all_options_convert_to_exact_seconds(self):
        self.assertEqual(
            [seconds_for_index(index) for index in range(len(BONUS_OPTIONS))],
            [900, 1800, 3600, 7200],
        )

    def test_invalid_option_is_rejected(self):
        with self.assertRaises(ValueError):
            seconds_for_index(-1)

    def test_settings_action_opens_the_advanced_configuration(self):
        launcher = mock.Mock()
        self.assertTrue(open_settings(launcher))
        launcher.assert_called_once_with(
            [SETUP_APPLICATION_PATH], start_new_session=True
        )

    def test_settings_action_reports_launch_failure(self):
        launcher = mock.Mock(side_effect=OSError("missing"))
        self.assertFalse(open_settings(launcher))

    def test_unconfigured_service_points_to_settings(self):
        message = unavailable_service_message(False)
        self.assertIn("ainda não foi configurado", message)
        self.assertIn("engrenagem", message)

    def test_configured_but_stopped_service_is_distinguished(self):
        message = unavailable_service_message(True)
        self.assertIn("serviço Compasso está indisponível", message)

    def test_missing_administrator_password_has_specific_message(self):
        message = friendly_error_name(
            "br.com.tempo.Agent.Error.PasswordNotConfigured"
        )
        self.assertIn("Nenhuma senha de administrador", message)
        self.assertIn("painel", message)

    def test_wrong_password_remains_distinct(self):
        self.assertEqual(
            friendly_error_name("br.com.tempo.Agent.Error.InvalidPassword"),
            "Senha incorreta.",
        )


if __name__ == "__main__":
    unittest.main()
