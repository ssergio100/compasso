#!/usr/bin/env python3
import json
import sys
import time

try:
    from gi.repository import GLib, Notify
    Notify.init("Tempo Alert Helper")
    HAS_GTK = True
except (ImportError, ValueError):
    HAS_GTK = False


def notify(title, message):
    if HAS_GTK:
        notification = Notify.Notification.new(title, message, None)
        notification.show()
    else:
        print(f"[ALERT] {title}: {message}")


def play_sound():
    if HAS_GTK:
        # Sound playback is intentionally minimal. The actual desktop sound
        # can be configured separately in a full GTK implementation.
        pass
    else:
        print("[ALERT SOUND]")


def handle_event(event):
    title = event.get("title", "Aviso de bloqueio")
    body = event.get("body", "Um bloqueio previsível está próximo.")
    notify(title, body)
    if event.get("sound_enabled", True):
        play_sound()


def main():
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        try:
            event = json.loads(line)
        except json.JSONDecodeError:
            print(f"invalid event: {line}")
            continue
        handle_event(event)


if __name__ == "__main__":
    main()
