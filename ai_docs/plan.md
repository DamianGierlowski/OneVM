# VM-Config Manager — Etapy Realizacji

## ETAP 1: Połączenie SSH

**Co robi:** Nawiązuje połączenie SSH z serwerem i wykonuje prostą komendę testową.

| | |
|---|---|
| **Input** | host, user, ścieżka do klucza SSH |
| **Output** | wynik komendy (np. hostname serwera) |
| **Test** | `vm-config ping --host 192.168.1.10 --user admin --key ~/.ssh/id_rsa` → zwraca nazwę hosta |

---

## ETAP 2: Transfer SFTP

**Co robi:** Pobiera i wysyła pliki między maszyną lokalną a serwerem przez SFTP.

| | |
|---|---|
| **Input** | połączenie SSH, ścieżka lokalna, ścieżka zdalna |
| **Output** | plik skopiowany (upload lub download) |
| **Test** | upload `test.txt`, download jako `test2.txt`, porównanie `diff test.txt test2.txt` → brak różnic |

---

## ETAP 3: Deploy z Backupem

**Co robi:** Normalizuje plik (CRLF→LF), pobiera backup obecnego pliku z serwera, wgrywa nową wersję.

| | |
|---|---|
| **Input** | plik lokalny, ścieżka docelowa na serwerze |
| **Output** | backup w `./backups/`, nowy plik na serwerze |
| **Test** | wgraj plik Windows (CRLF) → sprawdź `file` na serwerze (LF) → sprawdź że `./backups/` zawiera poprzednią wersję |

---

## ETAP 4: Manifest + Orchestracja

**Co robi:** Wczytuje manifest JSON, wykonuje deploy wszystkich zdefiniowanych plików, opcjonalnie restartuje usługi.

| | |
|---|---|
| **Input** | plik `servers.json` z definicją serwerów i plików |
| **Output** | wszystkie pliki wgrane, komendy restart wykonane |
| **Test** | `vm-config deploy --manifest servers.json` → pliki na serwerze zaktualizowane, `nginx -t` przechodzi |

---

## Podsumowanie

### Kolejność realizacji
```
ETAP 1 → ETAP 2 → ETAP 3 → ETAP 4
```

### Etap KRYTYCZNY
**ETAP 1 (Połączenie SSH)** — bez działającego SSH żadna inna funkcja nie zadziała.

### Można POMINĄĆ w v1
- **Rollback** — użytkownik może ręcznie skopiować backup
- **Dry-run** — można dodać jako `--dry-run` w ETAP 4 później
- **JSON output** — standardowy output wystarczy na start

---

**3 etapy działające > 5 zaplanowanych**

Etapy 1-3 dają działające narzędzie do wgrywania pojedynczych plików. Etap 4 dodaje wygodę manifestu i restarty.
