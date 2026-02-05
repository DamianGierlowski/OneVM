# Brief Biznesowy: VM-Config Manager

---

## 1. Problem

Administratorzy i zespoły DevOps regularnie aktualizują pliki konfiguracyjne na zdalnych serwerach. Obecne podejście jest:

- **Ryzykowne** — brak automatycznych kopii zapasowych przed zmianą prowadzi do sytuacji "zepsułem i nie mam jak wrócić"
- **Podatne na błędy** — różnice w formatowaniu plików między systemami (Windows vs Linux) powodują trudne do zdiagnozowania awarie
- **Manualne i czasochłonne** — każda zmiana wymaga serii powtarzalnych kroków (backup, upload, restart usługi)
- **Nieprzyjazne dla automatyzacji** — trudno zintegrować z narzędziami AI/agentami, które potrzebują strukturalnych odpowiedzi

---

## 2. Użytkownik

**Główny:** Administratorzy systemów, inżynierowie DevOps, programiści zarządzający własnymi serwerami.

**Drugorzędny:** Agenci AI i narzędzia automatyzacji (np. Claude Code, skrypty CI/CD).

**Kontekst użycia:** Użytkownik pracuje na stacji roboczej (Windows lub macOS) i zarządza jednym lub wieloma serwerami Linux.

---

## 3. MVP (Minimum Viable Product)

Narzędzie musi umożliwiać:

1. **Deploy** — wgranie plików konfiguracyjnych na serwer zgodnie z definicją w manifeście
2. **Automatyczny backup** — przed każdą zmianą pobierana jest kopia obecnego pliku z serwera (bez tego operacja nie może się wykonać)
3. **Rollback** — przywrócenie poprzedniej wersji pliku z lokalnego backupu
4. **Restart usług** — opcjonalne wykonanie komendy po wgraniu (np. przeładowanie serwisu)
5. **Lista zmian** — podgląd co zostanie zmienione bez faktycznej modyfikacji (tryb "na sucho")

---

## 4. Warunki Brzegowe

| Warunek | Opis |
|---------|------|
| **Backup obowiązkowy** | Jeśli nie można pobrać kopii istniejącego pliku — operacja musi zostać przerwana |
| **Kompatybilność systemów** | Narzędzie działa na Windows i macOS, wgrywa pliki na serwery Linux |
| **Normalizacja plików** | Automatyczna korekta formatowania (eliminacja problemów z końcami linii) |
| **Tryb AI-ready** | Możliwość uzyskania odpowiedzi w formacie strukturalnym (dla integracji z automatyzacją) |
| **Operacje uprzywilejowane** | Obsługa scenariuszy wymagających podwyższonych uprawnień na serwerze |
| **Single binary** | Jedno narzędzie bez zewnętrznych zależności — łatwa dystrybucja |

---

## 5. Technologia

### Dlaczego Golang

**Golang pasuje idealnie** — kompilacja do single binary bez zależności (wymaganie z briefu), natywne cross-compilation (Windows/macOS → wgrywanie na Linux), oraz oficjalna biblioteka SSH w rozszerzonym stdlib.

---

### Biblioteki

**STDLIB (preferowane):**

| Pakiet | Zastosowanie |
|--------|--------------|
| `os` | operacje plikowe, zmienne środowiskowe, ścieżki |
| `io` | kopiowanie strumieni, bufory |
| `path/filepath` | ścieżki cross-platform |
| `strings` | normalizacja końców linii (CRLF → LF) |
| `encoding/json` | manifest, strukturalny output (AI-ready) |
| `flag` | parsowanie argumentów CLI |
| `time` | timestampy dla nazw backupów |
| `fmt` | formatowanie komunikatów |
| `bytes` | porównywanie plików (diff) |

**ZEWNĘTRZNE (niezbędne):**

| Pakiet | Zastosowanie | Uzasadnienie |
|--------|--------------|--------------|
| `golang.org/x/crypto/ssh` | połączenie SSH, wykonanie komend | oficjalne rozszerzenie stdlib, jedyna opcja dla SSH |
| `github.com/pkg/sftp` | transfer plików przez SSH | stdlib nie ma SFTP, oszczędza ~200 linii własnej implementacji |

**Opcjonalnie:**

| Pakiet | Zastosowanie | Kiedy |
|--------|--------------|-------|
| `gopkg.in/yaml.v3` | manifest w YAML | jeśli JSON okaże się niewygodny dla użytkownika |

---

### Struktura plików (MVP)

```
vm-config/
├── main.go          # entry point, CLI parsing, orchestracja
├── manifest.go      # wczytywanie/walidacja manifestu
├── ssh.go           # połączenie SSH, wykonanie komend restart
├── transfer.go      # SFTP: upload, download (backup)
├── normalize.go     # konwersja CRLF → LF
├── backup.go        # zarządzanie lokalnymi backupami
├── go.mod
└── go.sum
```

---

### Jak uruchomić

```bash
# Deploy z manifestu
vm-config deploy --manifest servers.json --target production

# Rollback konkretnego pliku
vm-config rollback --file /etc/nginx/nginx.conf --server admin@192.168.1.10

# Dry-run (lista zmian bez modyfikacji)
vm-config deploy --manifest servers.json --dry-run

# Output strukturalny (dla AI/automatyzacji)
vm-config deploy --manifest servers.json --json

# Restart usługi po deploy
vm-config deploy --manifest servers.json --restart "systemctl reload nginx"
```

---

### Manifest (przykład JSON)

```json
{
  "servers": [
    {
      "host": "192.168.1.10",
      "user": "admin",
      "key": "~/.ssh/id_rsa"
    }
  ],
  "files": [
    {
      "local": "./configs/nginx.conf",
      "remote": "/etc/nginx/nginx.conf",
      "restart": "systemctl reload nginx"
    }
  ]
}
```
