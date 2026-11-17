// Persona 2 ITA - installer multipiattaforma (Windows / Linux / macOS), solo libreria standard Go.
// Crea una copia italiana della ISO originale applicando una patch a settori (formato P2IT v1).
package main

import (
	"bufio"
	"bytes"
	"compress/flate"
	"crypto/sha256"
	"embed"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
)

//go:embed payload
var payload embed.FS

type Variant struct {
	ID       string `json:"id"`
	Label    string `json:"label"`
	Patch    string `json:"patch"`
	OutName  string `json:"out_name"`
	OutSize  int64  `json:"out_size"`
	OutSHA   string `json:"out_sha256"`
	Textures bool   `json:"textures"` // offer the PPSSPP texture-pack install
	Note     string `json:"note"`
}

type Game struct {
	ID       string    `json:"id"`
	Title    string    `json:"title"`
	Serial   string    `json:"serial"`
	OrigSize int64     `json:"orig_size"`
	OrigSHA  string    `json:"orig_sha256"`
	OrigName string    `json:"orig_name"`
	Variants []Variant `json:"variants"`
	TexDir   string    `json:"tex_dir"` // payload dir copied into memstick/PSP/TEXTURES/<serial>
	TexNote  string    `json:"tex_note"`
}

type Manifest struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Games   []Game `json:"games"`
}

var (
	in  = bufio.NewReader(os.Stdin)
	man Manifest
	yes = flag.Bool("si", false, "rispondi si a tutte le domande (uso da script)")
)

func main() {
	outDir := flag.String("out", "", "cartella in cui scrivere le ISO italiane (default: accanto all'originale)")
	variant := flag.String("variante", "", "id variante (es. accenti, psp, tutte)")
	gameSel := flag.String("gioco", "", "gioco da tradurre senza menu: is | ep | tutti")
	flag.Usage = func() {
		fmt.Println("Uso: installa_persona2_ita [opzioni] [file.iso ...]")
		flag.PrintDefaults()
	}
	flag.Parse()
	b, err := payload.ReadFile("payload/manifest.json")
	if err != nil {
		fatal(err.Error())
	}
	if err := json.Unmarshal(b, &man); err != nil {
		fatal(err.Error())
	}
	fmt.Printf("==============================================\n %s  v%s\n==============================================\n", man.Name, man.Version)
	fmt.Println("Giochi supportati:")
	for _, g := range man.Games {
		fmt.Printf("  - %s [%s]\n", g.Title, g.Serial)
	}
	fmt.Println("\nQuesto programma NON modifica la tua ISO: ne crea una nuova copia in italiano.")

	// ISO passed on the command line / dragged onto the executable: auto-detect the game
	if len(flag.Args()) > 0 {
		ok := 0
		for _, p := range flag.Args() {
			if err := handleISO(cleanPath(p), nil, *outDir, *variant); err != nil {
				fmt.Println("ERRORE:", err)
			} else {
				ok++
			}
		}
		finish(ok > 0)
	}

	for {
		fmt.Println("\nQuale gioco vuoi tradurre?")
		for i, g := range man.Games {
			fmt.Printf("  %d) %s  [%s]\n", i+1, g.Title, g.Serial)
		}
		fmt.Printf("  %d) Tutti\n  0) Esci\n", len(man.Games)+1)
		var a string
		if *gameSel != "" {
			a = *gameSel
			for i, g := range man.Games {
				if strings.EqualFold(a, g.ID) {
					a = fmt.Sprint(i + 1)
				}
			}
			if strings.EqualFold(a, "tutti") {
				a = fmt.Sprint(len(man.Games) + 1)
			}
			*gameSel = "0"
		} else if *yes {
			fatal("con -si indica anche -gioco o un file .iso")
		} else {
			a = ask("Scelta: ")
		}
		var n int
		if _, err := fmt.Sscanf(a, "%d", &n); err != nil || n < 0 || n > len(man.Games)+1 {
			fmt.Println("Scelta non valida.")
			continue
		}
		if n == 0 {
			finish(true)
		}
		var games []Game
		if n == len(man.Games)+1 {
			games = man.Games
		} else {
			games = []Game{man.Games[n-1]}
		}
		for i := range games {
			g := games[i]
			fmt.Printf("\n========== %s ==========\n", g.Title)
			p := findISOFor(g)
			if p == "" {
				continue
			}
			if err := handleISO(p, &g, *outDir, *variant); err != nil {
				fmt.Println("ERRORE:", err)
			}
		}
		if *yes || *gameSel == "0" || !confirm("\nVuoi tradurre un altro gioco? (s/N): ", false) {
			finish(true)
		}
	}
}

func isoRank(p string, g Game) int {
	b := strings.ToLower(filepath.Base(p))
	switch {
	case b == strings.ToLower(g.OrigName):
		return 0
	case strings.Contains(b, "ita"):
		return 2
	}
	return 1
}

func finish(ok bool) {
	if ok {
		fmt.Println("\nFatto. Buon divertimento!")
	}
	pause()
	if ok {
		os.Exit(0)
	}
	os.Exit(1)
}

// findISOFor looks for the original ISO of g next to the program, else asks the user.
func findISOFor(g Game) string {
	var c []string
	for _, p := range scanISOs() {
		if fi, err := os.Stat(p); err == nil && fi.Size() == g.OrigSize {
			c = append(c, p)
		}
	}
	// most likely originals first: exact original file name, then names without "ITA"
	sort.SliceStable(c, func(i, j int) bool { return isoRank(c[i], g) < isoRank(c[j], g) })
	fmt.Printf("Serve la ISO originale: %s (%s)\n", g.OrigName, g.Serial)
	if len(c) > 0 {
		fmt.Println("Trovate:")
		for i, p := range c {
			fmt.Printf("  %d) %s\n", i+1, p)
		}
		a := ask("Numero [1], oppure trascina qui un altro file .iso: ")
		if a == "" {
			a = "1"
		}
		return pick(a, c)[0]
	}
	a := ask("Trascina qui la ISO originale (o scrivi il percorso) e premi Invio (vuoto = salta): ")
	return cleanPath(a)
}

func pick(a string, list []string) []string {
	a = strings.TrimSpace(a)
	if strings.EqualFold(a, "t") {
		return list
	}
	var n int
	if _, err := fmt.Sscanf(a, "%d", &n); err == nil && n >= 1 && n <= len(list) && !strings.ContainsAny(a, `/\.`) {
		return []string{list[n-1]}
	}
	return []string{cleanPath(a)}
}

func cleanPath(s string) string {
	s = strings.TrimSpace(s)
	s = strings.Trim(s, "\"'")
	if runtime.GOOS != "windows" {
		s = strings.ReplaceAll(s, `\ `, " ") // drag&drop in Linux/macOS terminals escapes spaces
	}
	if strings.HasPrefix(s, "~") {
		if h, err := os.UserHomeDir(); err == nil {
			s = filepath.Join(h, s[1:])
		}
	}
	return s
}

// scanISOs looks next to the executable and in the working dir (one level deep).
func scanISOs() []string {
	seen := map[string]bool{}
	var res []string
	dirs := []string{"."}
	if exe, err := os.Executable(); err == nil {
		dirs = append(dirs, filepath.Dir(exe))
	}
	sizes := map[int64]bool{}
	for _, g := range man.Games {
		sizes[g.OrigSize] = true
	}
	for _, d := range dirs {
		for _, pat := range []string{"*.iso", "*.ISO", "*/*.iso", "*/*.ISO"} {
			m, _ := filepath.Glob(filepath.Join(d, pat))
			for _, p := range m {
				ap, _ := filepath.Abs(p)
				if seen[ap] {
					continue
				}
				if fi, err := os.Stat(ap); err == nil && sizes[fi.Size()] {
					seen[ap] = true
					res = append(res, ap)
				}
			}
		}
	}
	sort.Strings(res)
	return res
}

func handleISO(p string, want *Game, outDir, variant string) error {
	fi, err := os.Stat(p)
	if err != nil {
		return fmt.Errorf("file non trovato: %s", p)
	}
	fmt.Printf("\n--- %s\n", p)
	var cands []Game
	for _, g := range man.Games {
		if g.OrigSize == fi.Size() {
			cands = append(cands, g)
		}
	}
	if len(cands) == 0 {
		for _, g := range man.Games {
			for _, v := range g.Variants {
				if v.OutSize == fi.Size() {
					if h, _ := fileSHA(p, "Controllo"); h == v.OutSHA {
						fmt.Printf("Questa ISO e' gia' la versione italiana di %s (%s).\n", g.Title, v.Label)
						return nil
					}
				}
			}
		}
		return errors.New(unknownMsg())
	}
	h, err := fileSHA(p, "Verifica ISO")
	if err != nil {
		return err
	}
	var game *Game
	for i := range cands {
		if cands[i].OrigSHA == h {
			game = &cands[i]
		}
	}
	if game == nil {
		for _, g := range man.Games {
			for _, v := range g.Variants {
				if v.OutSHA == h {
					return fmt.Errorf("questa e' gia' la versione italiana (%s) di %s: serve la ISO originale", v.Label, g.Title)
				}
			}
		}
		return errors.New("ISO non riconosciuta (checksum diverso).\n" + unknownMsg())
	}
	fmt.Printf("Riconosciuta: %s [%s]\n", game.Title, game.Serial)
	if want != nil && want.ID != game.ID {
		fmt.Printf("Nota: hai scelto %s ma questa ISO e' %s: procedo con %s.\n", want.Title, game.Title, game.Title)
	}

	vs := game.Variants
	if len(vs) > 1 {
		chosen := []Variant{}
		if variant == "" {
			fmt.Println("Versioni disponibili:")
			for i, v := range vs {
				fmt.Printf("  %d) %s\n", i+1, v.Label)
				if v.Note != "" {
					fmt.Printf("     %s\n", v.Note)
				}
			}
			fmt.Printf("  %d) tutte\n", len(vs)+1)
			variant = ask("Scelta [1]: ")
			if variant == "" {
				variant = "1"
			}
		}
		for i, v := range vs {
			if variant == fmt.Sprint(i+1) || strings.EqualFold(variant, v.ID) ||
				variant == fmt.Sprint(len(vs)+1) || strings.EqualFold(variant, "tutte") {
				chosen = append(chosen, v)
			}
		}
		if len(chosen) == 0 {
			return errors.New("scelta non valida")
		}
		vs = chosen
	}
	dir := outDir
	if dir == "" {
		dir = filepath.Dir(p)
	}
	for _, v := range vs {
		dst := filepath.Join(dir, v.OutName)
		created, err := build(p, dst, v)
		if err != nil {
			return err
		}
		if created {
			fmt.Printf("Creata: %s\n", dst)
		}
		if v.Textures && game.TexDir != "" {
			offerTextures(*game)
		}
	}
	return nil
}

func unknownMsg() string {
	s := "Servono le ISO originali non modificate:\n"
	for _, g := range man.Games {
		s += fmt.Sprintf("  - %s (%s), %d byte\n", g.OrigName, g.Serial, g.OrigSize)
	}
	return s
}

// ------------------------------------------------------------------ patching

func build(src, dst string, v Variant) (bool, error) {
	if h, err := quickSame(dst, v); err == nil && h {
		fmt.Println("La ISO italiana esiste gia' ed e' corretta:", dst)
		return false, nil
	}
	return true, buildISO(src, dst, v)
}

func buildISO(src, dst string, v Variant) error {
	pf, err := payload.Open("payload/" + v.Patch)
	if err != nil {
		return err
	}
	defer pf.Close()
	hdr := make([]byte, 20)
	if _, err := io.ReadFull(pf, hdr); err != nil || string(hdr[:4]) != "P2IT" {
		return errors.New("patch interna danneggiata")
	}
	outSize := int64(binary.LittleEndian.Uint64(hdr[8:16]))
	bs := int64(binary.LittleEndian.Uint32(hdr[16:20]))
	zr := flate.NewReader(bufio.NewReaderSize(pf, 1<<20))
	defer zr.Close()
	s, err := os.Open(src)
	if err != nil {
		return err
	}
	defer s.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	tmp := dst + ".parziale"
	o, err := os.Create(tmp)
	if err != nil {
		return err
	}
	w := bufio.NewWriterSize(o, 4<<20)
	hsh := sha256.New()
	mw := io.MultiWriter(w, hsh)
	var written int64
	prog := newProgress("Creazione ISO italiana", outSize)
	buf := make([]byte, bs*512)
	zero := make([]byte, bs)
	put := func(b []byte) error {
		if written+int64(len(b)) > outSize {
			b = b[:outSize-written]
		}
		n, err := mw.Write(b)
		written += int64(n)
		prog.set(written)
		return err
	}
	fail := func(e error) error { o.Close(); os.Remove(tmp); return e }
	op := make([]byte, 1)
	for {
		if _, err := io.ReadFull(zr, op); err != nil {
			return fail(errors.New("patch troncata"))
		}
		switch op[0] {
		case 0:
			goto done
		case 1:
			var a struct {
				Src uint64
				N   uint32
			}
			if err := binary.Read(zr, binary.LittleEndian, &a); err != nil {
				return fail(err)
			}
			off := int64(a.Src) * bs
			rem := int64(a.N) * bs
			for rem > 0 {
				n := rem
				if n > int64(len(buf)) {
					n = int64(len(buf))
				}
				k, err := s.ReadAt(buf[:n], off)
				if err != nil && err != io.EOF {
					return fail(err)
				}
				for i := k; i < int(n); i++ { // past end of original -> zero padding
					buf[i] = 0
				}
				if err := put(buf[:n]); err != nil {
					return fail(err)
				}
				off += n
				rem -= n
			}
		case 2:
			var n uint32
			if err := binary.Read(zr, binary.LittleEndian, &n); err != nil {
				return fail(err)
			}
			rem := int64(n) * bs
			for rem > 0 {
				c := rem
				if c > int64(len(buf)) {
					c = int64(len(buf))
				}
				if _, err := io.ReadFull(zr, buf[:c]); err != nil {
					return fail(errors.New("patch troncata"))
				}
				if err := put(buf[:c]); err != nil {
					return fail(err)
				}
				rem -= c
			}
		case 3:
			var n uint32
			if err := binary.Read(zr, binary.LittleEndian, &n); err != nil {
				return fail(err)
			}
			for i := uint32(0); i < n; i++ {
				if err := put(zero); err != nil {
					return fail(err)
				}
			}
		default:
			return fail(errors.New("patch interna danneggiata"))
		}
	}
done:
	prog.done()
	if err := w.Flush(); err != nil {
		return fail(err)
	}
	if err := o.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if written != outSize || hex.EncodeToString(hsh.Sum(nil)) != v.OutSHA {
		os.Remove(tmp)
		return errors.New("verifica della ISO creata fallita (spazio su disco esaurito?)")
	}
	fmt.Println("Verifica checksum: OK")
	os.Remove(dst)
	return os.Rename(tmp, dst)
}

func quickSame(dst string, v Variant) (bool, error) {
	fi, err := os.Stat(dst)
	if err != nil || fi.Size() != v.OutSize {
		return false, errors.New("no")
	}
	h, err := fileSHA(dst, "Controllo ISO esistente")
	return h == v.OutSHA, err
}

// ------------------------------------------------------------------ textures (PPSSPP)

func memstickCandidates() []string {
	home, _ := os.UserHomeDir()
	var c []string
	if exe, err := os.Executable(); err == nil {
		c = append(c, filepath.Join(filepath.Dir(exe), "memstick"))
	}
	switch runtime.GOOS {
	case "windows":
		docs := filepath.Join(home, "Documents")
		if od := os.Getenv("OneDrive"); od != "" {
			c = append(c, filepath.Join(od, "Documents", "PPSSPP"), filepath.Join(od, "Documenti", "PPSSPP"))
		}
		c = append(c, filepath.Join(docs, "PPSSPP"), filepath.Join(home, "Documenti", "PPSSPP"))
	case "darwin":
		c = append(c, filepath.Join(home, "Library", "Application Support", "PPSSPP"),
			filepath.Join(home, ".config", "ppsspp"))
	default:
		cfg := os.Getenv("XDG_CONFIG_HOME")
		if cfg == "" {
			cfg = filepath.Join(home, ".config")
		}
		c = append(c, filepath.Join(cfg, "ppsspp"),
			filepath.Join(home, ".var", "app", "org.ppsspp.PPSSPP", "config", "ppsspp"),
			filepath.Join(home, "snap", "ppsspp-emu", "current", ".config", "ppsspp"))
	}
	var found []string
	for _, p := range c {
		if st, err := os.Stat(filepath.Join(p, "PSP")); err == nil && st.IsDir() {
			found = append(found, filepath.Join(p, "PSP"))
		}
	}
	return found
}

func offerTextures(g Game) {
	fmt.Println("\nQuesta versione usa un font modificato per PPSSPP (texture replacement).")
	if g.TexNote != "" {
		fmt.Println(g.TexNote)
	}
	if !*yes && !confirm("Installare ora il font nella cartella di PPSSPP? (S/n): ", true) {
		fmt.Println("Saltato. Puoi rilanciare il programma piu' tardi.")
		return
	}
	cands := memstickCandidates()
	target := ""
	if len(cands) == 1 && (*yes || confirm(fmt.Sprintf("Trovata la cartella PSP di PPSSPP:\n  %s\nUsarla? (S/n): ", cands[0]), true)) {
		target = cands[0]
	} else {
		for i, c := range cands {
			fmt.Printf("  %d) %s\n", i+1, c)
		}
		a := ask("Numero, oppure trascina qui la cartella 'PSP' della memory stick di PPSSPP (vuoto = salta): ")
		if a == "" {
			return
		}
		var n int
		if _, err := fmt.Sscanf(a, "%d", &n); err == nil && n >= 1 && n <= len(cands) {
			target = cands[n-1]
		} else {
			target = cleanPath(a)
			if filepath.Base(target) != "PSP" {
				if st, err := os.Stat(filepath.Join(target, "PSP")); err == nil && st.IsDir() {
					target = filepath.Join(target, "PSP")
				}
			}
		}
	}
	dst := filepath.Join(target, "TEXTURES", g.Serial)
	n, err := copyTree("payload/"+g.TexDir, dst)
	if err != nil {
		fmt.Println("ERRORE copia texture:", err)
		return
	}
	fmt.Printf("Copiati %d file in %s\n", n, dst)
	if _, err := os.Stat(filepath.Join(dst, "textures.ini")); err != nil {
		fmt.Println("ATTENZIONE: in quella cartella manca textures.ini: installa prima il pack HD di Karasu")
		fmt.Println("(vedi LEGGIMI), poi rilancia questo programma.")
	}
	fmt.Println("In PPSSPP attiva: Impostazioni > Grafica > Sostituisci texture.")
}

func copyTree(src, dst string) (int, error) {
	n := 0
	err := walkEmbed(src, func(p string, data []byte) error {
		rel := strings.TrimPrefix(p, src+"/")
		out := filepath.Join(dst, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(out), 0o755); err != nil {
			return err
		}
		if old, err := os.ReadFile(out); err == nil && !bytes.Equal(old, data) {
			bak := out + ".orig"
			if _, e := os.Stat(bak); e != nil {
				os.WriteFile(bak, old, 0o644)
			}
		}
		n++
		return os.WriteFile(out, data, 0o644)
	})
	return n, err
}

func walkEmbed(dir string, fn func(string, []byte) error) error {
	ents, err := payload.ReadDir(dir)
	if err != nil {
		return err
	}
	for _, e := range ents {
		p := dir + "/" + e.Name()
		if e.IsDir() {
			if err := walkEmbed(p, fn); err != nil {
				return err
			}
			continue
		}
		b, err := payload.ReadFile(p)
		if err != nil {
			return err
		}
		if err := fn(p, b); err != nil {
			return err
		}
	}
	return nil
}

// ------------------------------------------------------------------ utils

type progress struct {
	label string
	total int64
	last  time.Time
	start time.Time
}

func newProgress(l string, t int64) *progress {
	return &progress{label: l, total: t, start: time.Now()}
}
func (p *progress) set(v int64) {
	if time.Since(p.last) < 250*time.Millisecond && v < p.total {
		return
	}
	p.last = time.Now()
	pct := float64(v) * 100 / float64(p.total)
	bar := strings.Repeat("#", int(pct/4)) + strings.Repeat(".", 25-int(pct/4))
	fmt.Printf("\r%s [%s] %5.1f%%", p.label, bar, pct)
}
func (p *progress) done() {
	p.set(p.total)
	fmt.Printf("  (%.0fs)\n", time.Since(p.start).Seconds())
}

func fileSHA(p, label string) (string, error) {
	f, err := os.Open(p)
	if err != nil {
		return "", err
	}
	defer f.Close()
	fi, _ := f.Stat()
	h := sha256.New()
	pr := newProgress(label, fi.Size())
	buf := make([]byte, 4<<20)
	var done int64
	for {
		n, err := f.Read(buf)
		if n > 0 {
			h.Write(buf[:n])
			done += int64(n)
			pr.set(done)
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	pr.done()
	return hex.EncodeToString(h.Sum(nil)), nil
}

func ask(p string) string {
	if *yes {
		fmt.Println(p)
		return ""
	}
	fmt.Print(p)
	s, _ := in.ReadString('\n')
	return strings.TrimSpace(s)
}
func confirm(p string, def bool) bool {
	a := strings.ToLower(ask(p))
	if a == "" {
		return def
	}
	return a == "s" || a == "si" || a == "sì" || a == "y" || a == "yes"
}
func pause() {
	if *yes {
		return
	}
	if fi, _ := os.Stdin.Stat(); fi != nil && fi.Mode()&os.ModeCharDevice != 0 {
		ask("\nPremi Invio per chiudere...")
	}
}
func fatal(m string) { fmt.Println("ERRORE:", m); pause(); os.Exit(1) }
