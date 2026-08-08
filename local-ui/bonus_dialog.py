#!/usr/bin/env python3
"""GTK 4 dialog for granting local Compasso bonus time."""

import gi

gi.require_version("Gtk", "4.0")
from gi.repository import Gio, GLib, Gtk


BUS_NAME = "br.com.tempo.Agent"
OBJECT_PATH = "/br/com/tempo/Agent"
INTERFACE_NAME = "br.com.tempo.Agent"
BONUS_OPTIONS = (15, 30, 60, 120)


def seconds_for_index(index):
    if index < 0 or index >= len(BONUS_OPTIONS):
        raise ValueError("invalid bonus selection")
    return BONUS_OPTIONS[index] * 60


def friendly_error(error):
    remote_name = Gio.DBusError.get_remote_error(error) or ""
    if remote_name.endswith("InvalidPassword"):
        return "Senha incorreta."
    if remote_name.endswith("RateLimited"):
        return "Muitas tentativas. Aguarde um pouco e tente novamente."
    return "Não foi possível adicionar o tempo. Verifique se o Compasso está ativo."


class BonusWindow(Gtk.ApplicationWindow):
    def __init__(self, application):
        super().__init__(application=application, title="Adicionar tempo — Compasso")
        self.set_default_size(420, 260)
        self.set_resizable(False)
        self.proxy = None

        content = Gtk.Box(orientation=Gtk.Orientation.VERTICAL, spacing=12)
        content.set_margin_top(24)
        content.set_margin_bottom(24)
        content.set_margin_start(24)
        content.set_margin_end(24)
        self.set_child(content)

        title = Gtk.Label(label="Adicionar tempo de uso")
        title.add_css_class("title-2")
        title.set_halign(Gtk.Align.START)
        content.append(title)

        description = Gtk.Label(label="Escolha o tempo e informe a senha do responsável.")
        description.set_wrap(True)
        description.set_halign(Gtk.Align.START)
        content.append(description)

        self.duration = Gtk.DropDown.new_from_strings(
            [f"{minutes} minutos" for minutes in BONUS_OPTIONS]
        )
        self.duration.set_selected(1)
        content.append(self.duration)

        self.password = Gtk.Entry()
        self.password.set_placeholder_text("Senha do responsável")
        self.password.set_visibility(False)
        self.password.set_input_purpose(Gtk.InputPurpose.PASSWORD)
        self.password.connect("activate", self._submit)
        content.append(self.password)

        self.submit = Gtk.Button(label="Adicionar tempo")
        self.submit.add_css_class("suggested-action")
        self.submit.connect("clicked", self._submit)
        content.append(self.submit)

        self.status = Gtk.Label()
        self.status.set_wrap(True)
        self.status.set_halign(Gtk.Align.START)
        content.append(self.status)

        try:
            self.proxy = Gio.DBusProxy.new_for_bus_sync(
                Gio.BusType.SYSTEM,
                Gio.DBusProxyFlags.NONE,
                None,
                BUS_NAME,
                OBJECT_PATH,
                INTERFACE_NAME,
                None,
            )
            if self.proxy.get_name_owner() is None:
                self.proxy = None
                self.status.set_text("O serviço Compasso não está disponível.")
                self.submit.set_sensitive(False)
        except GLib.Error:
            self.status.set_text("O serviço Compasso não está disponível.")
            self.submit.set_sensitive(False)

    def _submit(self, _widget):
        if self.proxy is None or not self.submit.get_sensitive():
            return
        password = self.password.get_text()
        if not password:
            self.status.set_text("Informe a senha do responsável.")
            return
        try:
            seconds = seconds_for_index(self.duration.get_selected())
        except ValueError:
            self.status.set_text("Escolha um período válido.")
            return

        self.submit.set_sensitive(False)
        self.status.set_text("Verificando…")
        self.proxy.call(
            "AddLocalBonus",
            GLib.Variant("(su)", (password, seconds)),
            Gio.DBusCallFlags.NONE,
            15_000,
            None,
            self._completed,
            seconds,
        )

    def _completed(self, proxy, result, seconds):
        self.password.set_text("")
        self.submit.set_sensitive(True)
        try:
            response = proxy.call_finish(result)
            event_uuid = response.unpack()[0]
            if not event_uuid:
                raise ValueError("empty event UUID")
            self.status.set_text(f"Tempo adicionado: {seconds // 60} minutos.")
        except GLib.Error as error:
            self.status.set_text(friendly_error(error))
        except (TypeError, ValueError):
            self.status.set_text("O agente retornou uma resposta inválida.")


class BonusApplication(Gtk.Application):
    def __init__(self):
        super().__init__(application_id="br.com.tempo.LocalBonus")

    def do_activate(self):
        window = self.props.active_window
        if window is None:
            window = BonusWindow(self)
        window.present()


def main():
    return BonusApplication().run(None)


if __name__ == "__main__":
    raise SystemExit(main())
