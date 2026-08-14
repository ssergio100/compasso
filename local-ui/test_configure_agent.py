import unittest
from unittest import mock

from configure_agent import (
    DEFAULT_SERVER_URL,
    available_controlled_users,
    controlled_user_confirmation_text,
    initial_controlled_user,
    run_privileged_configuration,
    validate_form,
)


class AgentSetupLogicTest(unittest.TestCase):
    def test_default_server_uses_https(self):
        self.assertEqual(DEFAULT_SERVER_URL, "https://apicompasso.smresume.com")

    def test_complete_https_form_is_valid(self):
        self.assertIsNone(
            validate_form(
                controlled_user="child",
                server_url="https://example.test",
                device_id="device-1",
                device_token="secret-token",
            )
        )

    def test_remote_http_is_rejected(self):
        message = validate_form(
            controlled_user="child",
            server_url="http://192.168.1.10:8181",
            device_id="device-1",
            device_token="secret-token",
        )
        self.assertIn("HTTPS", message)

    def test_missing_device_credentials_are_rejected(self):
        self.assertIn(
            "device_id",
            validate_form("child", "https://example.test", "", "secret-token"),
        )

    def test_confirmation_identifies_the_selected_account(self):
        self.assertIn("sergio", controlled_user_confirmation_text("sergio"))
        self.assertIn("Escolha", controlled_user_confirmation_text(""))
        self.assertIn(
            "device_token",
            validate_form("child", "https://example.test", "device-1", ""),
        )

    def test_first_available_user_is_the_actual_initial_selection(self):
        self.assertEqual(initial_controlled_user(["sergio", "child"]), "sergio")
        self.assertEqual(initial_controlled_user([]), "")

    @mock.patch("configure_agent.subprocess.run")
    def test_rejected_credentials_are_explained(self, run):
        run.return_value = mock.Mock(
            returncode=1,
            stderr="tempo-agent-configure: server rejected device credentials: heartbeat returned HTTP 401",
        )
        message = run_privileged_configuration(
            {
                "controlled_user": "sergio",
                "server_url": "https://example.test",
                "device_id": "device",
                "device_token": "secret",
            }
        )
        self.assertIn("recusou", message)
        self.assertIn("nova credencial", message)

    @mock.patch("configure_agent.getpass.getuser", return_value="sergio")
    @mock.patch("configure_agent.pwd.getpwall")
    def test_user_list_prioritizes_current_regular_user(self, getpwall, _getuser):
        getpwall.return_value = [
            mock.Mock(pw_uid=0, pw_name="root", pw_shell="/bin/bash"),
            mock.Mock(pw_uid=1001, pw_name="child", pw_shell="/bin/bash"),
            mock.Mock(pw_uid=1000, pw_name="sergio", pw_shell="/bin/bash"),
            mock.Mock(pw_uid=65534, pw_name="nobody", pw_shell="/usr/sbin/nologin"),
        ]
        self.assertEqual(available_controlled_users(), ["sergio", "child"])


if __name__ == "__main__":
    unittest.main()
