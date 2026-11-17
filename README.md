# Persona 2 ITA (PSP)

Patch di traduzione italiana **non ufficiale** per i due capitoli di Persona 2 su PSP:

| Gioco | Versione richiesta | Serial |
|---|---|---|
| **Persona 2: Innocent Sin** | Europea | ULES01557 |
| **Persona 2: Eternal Punishment** (ペルソナ2 罰) | Giapponese | ULJM06081 |

Il repository contiene **solo la patch e l'installer**: il gioco non è incluso. Ti serve una copia della ISO originale ricavata dal tuo UMD.

---

## Installazione

1. Scarica dalla pagina **Releases** l'installer per il tuo sistema:

   | Sistema | File |
   |---|---|
   | Windows (64 bit) | `installa_persona2_ita_windows_amd64.exe` |
   | Windows ARM | `installa_persona2_ita_windows_arm64.exe` |
   | Linux / Steam Deck | `installa_persona2_ita_linux_amd64` |
   | Linux ARM | `installa_persona2_ita_linux_arm64` |
   | macOS Apple Silicon | `installa_persona2_ita_macos_arm64` |
   | macOS Intel | `installa_persona2_ita_macos_amd64` |

2. Mettilo nella stessa cartella della ISO originale e avvialo. In alternativa trascina la ISO sopra il programma, oppure dentro la finestra quando te la chiede.
3. Il programma riconosce il gioco dal checksum. Poi crea **una nuova ISO in italiano** accanto all'originale e ne verifica il checksum. La ISO originale non viene toccata.

Su Linux e macOS rendi eseguibile il file con `chmod +x installa_persona2_ita_*`. Su macOS, la prima volta usa *clic destro → Apri*, perché il programma non è firmato.

Uso da riga di comando:

```
installa_persona2_ita [-variante accenti|psp|tutte] [-out cartella] [-si] file.iso ...
```

### Innocent Sin: due versioni

- **Con accenti (PPSSPP)**: testi con à è é ì ò ù. Funziona su PPSSPP con il texture replacement:
  1. Installa il pack *HD UI for Persona 2 – Innocent Sin* di Karasu ([GameBanana](https://gamebanana.com/mods/308752)). La cartella `ULUS10584` va rinominata `ULES01557` dentro `memstick/PSP/TEXTURES/`.
  2. Lascia che l'installer copi sopra il font modificato.
  3. In PPSSPP attiva *Impostazioni → Grafica → Sostituisci texture*.
- **PSP**: senza accenti (perche', citta'), per PSP/PS Vita reali o emulatori senza texture pack.

### Eternal Punishment

Una sola versione, con accenti nativi nel font del gioco. Funziona su PSP, PS Vita e PPSSPP.

---

## Cosa è tradotto

**Eternal Punishment**
- Tutti i dialoghi, gli eventi e le scene, scenario di Tatsuya compreso.
- I Contatti con i demoni.
- Le dicerie.
- I menu, le descrizioni di oggetti e abilità.
- I sottotitoli dei filmati.
- Le immagini di testo: avvisi iniziali, menu grafici, negozi, Velvet Room, risultati di battaglia, finale.

Restano invariati:
- I nomi propri: Persona, demoni, abilità, Arcani, luoghi.
- La schermata del titolo.

**Innocent Sin**
- Tutti i testi di gioco e dei menu.
- Restano in inglese: i testi grafici (immagini dei menu), il Climax Theater e poche righe che non entravano nello spazio disponibile.

## Avvertenze

- **Traduzione automatica** (Gemini / Claude), partendo dalle traduzioni inglesi e con glossario e controlli automatici su tag, lunghezza righe e caratteri. Non è stata revisionata da un traduttore umano: refusi e frasi poco naturali sono possibili. Segnalazioni e correzioni sono benvenute tramite *Issues* o *Pull request*.
- In Innocent Sin non usare "Install Data" dal menu del gioco con la ISO originale: reinstalla i file originali.
- Testata su PPSSPP 1.20.

## Struttura del repository

```
installer/          sorgente Go dell'installer (solo libreria standard)
  main.go
  payload/          manifest.json + font PPSSPP; le patch .p2it NON sono nel repo
tools/
  mkpatch.py        genera le patch .p2it (diff a settori da 2048 byte, DEFLATE)
  build.sh          compila l'installer per tutte le piattaforme in dist/
  patch_info/       dimensioni e SHA-256 di ISO originali e tradotte
```

Formato `.p2it`: la ISO viene ricostruita a blocchi da 2048 byte. Ogni blocco o viene copiato dalla ISO originale, o è un dato nuovo compresso, o è una sequenza di zeri. Alla fine il risultato viene verificato con SHA-256.

### Compilare

Richiede Go ≥ 1.21.

1. Scarica dalla Release i file `ep_ita.p2it`, `is_ita_accenti.p2it` e `is_ita_psp.p2it`. Sono troppo grandi per il repository.
2. Mettili in `installer/payload/`.
3. Lancia:

```
bash tools/build.sh        # oppure: GO=/percorso/go bash tools/build.sh
```

Per rigenerare una patch partendo da una ISO tradotta:

```
python tools/mkpatch.py ORIGINALE.iso TRADOTTA.iso installer/payload/nome.p2it
```

Poi aggiorna dimensioni e SHA-256 in `installer/payload/manifest.json`.

## Crediti

- **Eternal Punishment**: si basa sulla traduzione inglese per PSP di *eiowlta, Sayucchi, DarTisD*: [P2-EP-PSP](https://github.com/sayucchin/P2-EP-PSP), [p2_tool](https://github.com/eiowlta/p2_tool), [p2ep_template](https://github.com/eiowlta/p2ep_template). Lo script dello scenario di Tatsuya viene dal progetto *p2px-scenario*.
- **Innocent Sin**: strumenti e struttura dei testi del progetto francese [Grimoire du Coeur – P2-IS-FR-PSP](https://github.com/Grimoire-du-Coeur/P2-IS-FR-PSP) (CC BY-NC-SA 4.0).
- **Font HD di Innocent Sin**: Karasu (*HD UI for Persona 2*), glifi modificati per ì ò È.
- Traduzione italiana e installer: Daniele Dituri.

## Licenza

La patch è distribuita con licenza **CC BY-NC-SA 4.0** (uso non commerciale, stessa licenza), come il progetto francese da cui deriva la parte di Innocent Sin.

*Persona 2* © ATLUS / SEGA. Progetto amatoriale senza scopo di lucro, non affiliato ad Atlus. Non contiene né distribuisce il gioco: compra l'originale.
