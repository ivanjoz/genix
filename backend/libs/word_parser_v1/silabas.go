package libs

import "strings"

var silabas = map[uint8][]string{
	1: { "1" },
	2: { "2" },
	3: { "3", "y" }, // 'y' is here
	4: { "4", "-" },
	5: { "5" },
	6: { "6", "z" }, // 'z' is here
	7: { "7", "x" }, // 'x' is here
	8: { "8", "w" }, // 'w' is here
	9: { "9" },
	203: { "sa", "xa" },
	200: { "se", "ce", "xe" },
	201: { "si", "ci", "xi" },
	202: { "so", "xo" },
	205: { "su", "xu" },
	220: { "ca", "ka", "qa" },
	221: { "co", "ko", "qo" },
	222: { "cu", "ku"},
	209: { "g", "gr", "grs" },
	210: { "m", "ml" },
	211: { "k", "kg", "kgs" },
	212: { "ud", "un", "uns" },
	213: { "hom", "cm" },
}

var vocales = []string{"a","e","i","o","u"}
var consonants = []string{"b","c","d","f","g","h","j","k","l","m","n","ñ","p","q","r","s","t","v"} // w,x,y,z are in 1-9
var conectores = []string{"de","la","con","para","en"}

// Se autocompletan
// * significa que se combinan con vocales
// - se combinan con vocalesMenosU
var combinacionesConVocales = [][]string{
	{ "b*", "v*" }, { "k-" }, { "d*" }, { "f*" }, { "g*", "gu*" }, { "h*" },
	{ "j*" }, { "l*" },{ "m*" }, { "m*r", "m*l", "m*c" }, { "n*" },  { "ñ-" }, 
	{ "p*" }, { "p*r", "p*l" },
 	{ "q*", "qu*" }, { "ch*" }, { "ll*" }, { "-l" },{ "-n" }, { "r*", "rr*" },
}

var frecuentes = []string{
	"au", "ia", "io", "ua", "ue", "ie", "ai", "ei", "eu", "oi",
	"ou", "iu", "ui", "te", "ta", "to", "ra", "ti", "ro", "re",
	"men", "dor", "cion", "es", "en", "ri", "ar", "za", "des", "in",
	"con", "mien", "ble", "tar", "car", "al", "zo", "tra", "tu", "rio",
	"dad", "cia", "nar", "zar", "an", "tro", "lar", "gra", "rra", "cio",
	"rre", "pro", "em", "pre", "su", "gua", "per", "tri", "gar", "cer",
	"rro", "ter", "mar", "can", "tre", "man", "cor", "cen", "bre", "tor",
	"ria", "par", "nis", "rar", "bar", "ex", "lis", "ver", "dis", "bra",
	"ru", "ten", "sar", "tis", "dar", "com", "sis", "llar", "sen", "nal",
	"gue", "cal", "rri", "for", "tan", "jar", "ya", "dia", "den", "im",
	"char", "ris", "pla", "ven", "res", "len", "lan", "cas", "nes", "por",
	"dio", "ren", "san", "ran", "sal", "cua", "gui", "as", "tras", "tal",
	"cla", "nia", "pen", "tes", "ras", "cre", "cos", "sio", "les", "pan",
	"sion", "ton", "bri", "cro", "dro", "bro", "tas", "mor", "pia", "or",
	"yo", "tos", "ber", "mal", "nio", "gre", "gan", "ral", "mon", "cri",
	"lia", "cul", "cam", "lec", "nan", "dra", "sor", "ban", "zu", "dri",
	"mos", "ple", "sub", "guar", "der", "sin", "pli", "gen", "pos", "var",
	"ser", "rru", "mis", "bal", "pas", "lin", "nos", "pri",
}

func makeSilabas(){
	used := make(map[string]bool)
	for _, v := range silabas {
		for _, s := range v {
			used[s] = true
		}
	}
	
	vowelIndex := 0
	consIndex := 0
	combinationGroupIndex := 0
	combinationVowelIndex := 0
	frequentIndex := 0
	
	getNextCombination := func() ([]string, bool) {
		for combinationGroupIndex < len(combinacionesConVocales) {
			group := combinacionesConVocales[combinationGroupIndex]
			if combinationVowelIndex >= len(vocales) {
				combinationGroupIndex++
				combinationVowelIndex = 0
				continue
			}
			
			vowel := vocales[combinationVowelIndex]
			combinationVowelIndex++
			
			var groupSyllables []string
			for _, pattern := range group {
				if len(pattern) < 2 { continue }
				
				suffix := pattern[len(pattern)-1:]
				var root string
				
				isVowelFirst := false
				if strings.HasPrefix(pattern, "-") && len(pattern) > 1 {
					isVowelFirst = true
					root = pattern[1:]
				}

				allowed := false
				if suffix == "*" {
					allowed = true
				} else if suffix == "-" {
					if combinationVowelIndex-1 != 4 { // index 4 is 'u'
						allowed = true
					}
				} else if isVowelFirst {
					allowed = true
				}

				if allowed {
					var s string
					if isVowelFirst {
						s = vowel + root
					} else {
						if !isVowelFirst {
							root = pattern[:len(pattern)-1]
						}
						s = root + vowel
					}
					
					if !used[s] {
						groupSyllables = append(groupSyllables, s)
						used[s] = true
					}
				}
			}
			
			if len(groupSyllables) > 0 {
				return groupSyllables, true
			}
		}
		return nil, false
	}

	for id := uint8(10); id < 255; id++ { 
		if _, exists := silabas[id]; exists {
			continue
		}
		
		// Priority 1: Vowels
		if vowelIndex < len(vocales) {
			s := vocales[vowelIndex]
			vowelIndex++
			if !used[s] {
				silabas[id] = []string{s}
				used[s] = true
				continue
			}
		}

		// Priority 2: Single Consonants (to ensure full coverage)
		if consIndex < len(consonants) {
			s := consonants[consIndex]
			consIndex++
			if !used[s] {
				silabas[id] = []string{s}
				used[s] = true
				continue
			} else {
				// If used, try next
				id-- // retry this ID with next consonant
				continue
			}
		}

		// Priority 3: Combinations
		combos, ok := getNextCombination()
		if ok {
			silabas[id] = combos
			continue
		}

		// Priority 4: Frequents
		if frequentIndex < len(frecuentes) {
			for frequentIndex < len(frecuentes) {
				s := frecuentes[frequentIndex]
				frequentIndex++
				if !used[s] {
					silabas[id] = []string{s}
					used[s] = true
					goto nextID
				}
			}
		}
		break
		nextID:
	}
}
