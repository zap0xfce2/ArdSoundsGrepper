# ArdSoundsGrepper

Lädt Folgen von [ARD-Sounds](https://www.ardsounds.de)-Sendungen als MP3 – interaktiv per Terminal-Oberfläche (TUI) oder automatisch per `sync` für abonnierte Sendungen. Bereits vorhandene Folgen werden erkannt, geladen wird nur, was fehlt.

> Nur zur privaten Nutzung – die Inhalte sind urheberrechtlich geschützt.

## KI-Assistierte Entwicklung
> [!IMPORTANT]
> Dieses Projekt wurde mit erheblicher KI-Unterstützung erstellt, und das ist beabsichtigt und transparent.
>
> Teile der Codebasis wurden mit KI generiert, aber die Anwendung wurde nicht als ungeprüfter Output herausgegeben. Der generierte Code wurde von einem menschlichen Entwickler überprüft, korrigiert und validiert, bevor er veröffentlicht wurde.

## Build

Voraussetzungen: Go ≥ 1.27, [go-task](https://taskfile.dev).

```bash
task          # baut bin/ardsoundsgrepper (Version vYYMMDDhhmm = Build-Zeitpunkt)
task test     # alle Tests (offline)
task lint     # shellcheck, go vet, gofmt
task smoke    # Live-Test gegen die echte API (lädt eine Folge in ein Temp-Verzeichnis)
```

## TUI

```bash
./bin/ardsoundsgrepper
./bin/ardsoundsgrepper --version
```

Mit mindestens einem Abo startet die TUI im Abo-Tab (neue Folgen werden gleich gezählt), sonst in der Suche mit Cursor im Suchfeld. `?` zeigt alle Tasten in der App.

| Wo | Tasten |
|---|---|
| überall | `1` `2` `3` / `←` `→` / `Tab` Reiter wechseln · `?` Hilfe · `c` Config anzeigen · `q` / `ctrl+c` beenden (Rückfrage, wenn Downloads laufen) |
| Suche | Suchbegriff, Sendungs-URL oder URN tippen + `Enter` · `↑↓` wählen · `Enter` öffnen · `a` abonnieren · `i` neue Suche |
| Folgen | `Enter` lädt markierte Folgen, sonst die unter dem Cursor · `Leertaste` markieren · `A` alle fehlenden · `/` filtern · `t` Trailer ein/aus · `o` Zielordner · `Esc` zurück |
| Pfad-Dialog | `↑↓` letzte 10 Pfade · `Enter` übernehmen · `Tab` Eintrag bearbeiten · `n` neuer Pfad · `x` aus der Liste entfernen · `Esc` schließen |
| Downloads | `↑↓` wählen · `x` abbrechen · `r` erneut versuchen |
| Abos | `Enter` öffnen · `o` Zielordner des Abos ändern · `s` alle prüfen & laden · `r` neu zählen · `d` entfernen |

Zielordner werden bei Bedarf angelegt. Ein Abo hat einen festen Zielordner; lädt man eine Folge spontan woanders hin, bleibt er unverändert.

## Automatisch: `sync`

```bash
./bin/ardsoundsgrepper sync                    # alle Abos
./bin/ardsoundsgrepper sync --abo <URN|Titel>  # nur ein Abo (eindeutiger Titelteil genügt)
```

Lädt ohne Rückfragen und ohne Terminal nur die fehlenden Folgen in den Zielordner des jeweiligen Abos. Eine Sperrdatei verhindert, dass zwei Läufe (oder `sync` und TUI) gleichzeitig laden.

| Exit-Code | Bedeutung |
|---|---|
| `0` | alles ok (auch: nichts Neues) |
| `1` | mindestens ein Download oder Abo fehlgeschlagen |
| `2` | falscher Aufruf, ungültige Config, `--abo` passt auf kein/mehrere Abos, läuft bereits |

Beispiel-Crontab (täglich 6 Uhr):

```cron
0 6 * * * /pfad/zu/bin/ardsoundsgrepper sync >> ~/ardsoundsgrepper.log 2>&1
```

## Config & Dateien

| Datei | Inhalt |
|---|---|
| `$XDG_CONFIG_HOME/ardsoundsgrepper/config.toml` (Standard `~/.config/…`) | Einstellungen und Abos – von Hand editierbar |
| `$XDG_STATE_HOME/ardsoundsgrepper/history.json` (Standard `~/.local/state/…`) | zuletzt genutzte Zielordner (max. 10) |
| `$XDG_STATE_HOME/ardsoundsgrepper/sync.lock` | Sperre gegen parallele Läufe |

```toml
target_dir      = "~/Music/ARD Sounds"   # Standard-Basis; je Sendung ein Unterordner
include_trailer = false

[[subscriptions]]
urn        = "urn:ard:show:daebe2a39366d28a"
title      = "Mimi Sandmädchen"
target_dir = "~/Music/Mimi"              # optional, sonst <target_dir>/<Titel>
```

Dateinamen: `YYYY-MM-DD - <Titel>.mp3`. Downloads laufen über eine `.part`-Datei und werden erst nach erfolgreicher Prüfung umbenannt – es entstehen keine halben MP3s.