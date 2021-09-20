package cff

type mainIndex int

// Index tables
const (
	NameIndex mainIndex = iota
	DictIndex
	StringIndex
	GlobalSubrIndex
	CharSet
	Encoding
	CharStringsIndex
	PrivateDict
	LocalSubrsIndex
)

func (mi mainIndex) String() string {
	switch mi {
	case NameIndex:
		return "NameIndex"
	case DictIndex:
		return "DictIndex"
	case StringIndex:
		return "StringIndex"
	case GlobalSubrIndex:
		return "GlobalSubrIndex"
	case CharSet:
		return "CharSet"
	case Encoding:
		return "Encoding"
	case CharStringsIndex:
		return "CharStringsIndex"
	case PrivateDict:
		return "PrivateDict"
	case LocalSubrsIndex:
		return "LocalSubrsIndex"
	}
	return ""
}

// CFF represents a CFF font file
type CFF struct {
	Major           uint8
	Minor           uint8
	HdrSize         uint8
	Fontindex       int
	offsetSize      uint8
	fontnames       []string
	globalSubrIndex [][]byte
	Font            []*Font
	strings         []string
	stringToInt     map[string]int
}

// SID represents a CFF string
type SID int

// Font is a CFF font instance. There can be more than one CFF font in a font file.
type Font struct {
	global             *CFF
	name               SID
	bbox               []int
	bluefuzz           int
	bluescale          float64
	blueshift          int
	bluevalues         []int
	charset            []SID
	charsetOffset      int64
	charsetFormat      uint8
	charstringsOffset  int64
	CharStrings        [][]byte
	cidcount           int
	copyright          SID
	data               []byte // for font writing
	defaultWidthX      int
	dict               []byte
	encodingOffset     int
	encoding           map[int]int
	encodingFormat     uint8
	familyblues        []int
	familyotherblues   []int
	fdarray            int64
	fdselect           int64
	fullname           SID
	familyname         SID
	initialRandomSeed  int
	nominalWidthX      int
	notice             SID
	ordering           SID
	otherblues         []int
	privatedictoffset  int64
	privatedictsize    int
	privatedict        []byte
	registry           SID
	stdhw              int
	stdvw              int
	stemsnaph          []int
	stemsnapv          []int
	subrsOffset        int
	subrsIndex         [][]byte
	supplement         int
	uniqueid           int
	underlinePosition  float64
	underlineThickness float64
	version            SID
	weight             SID
}

var predefinedStrings = []string{".notdef",
	"space",
	"exclam",
	"quotedbl",
	"numbersign",
	"dollar",
	"percent",
	"ampersand",
	"quoteright",
	"parenleft",
	"parenright",
	"asterisk",
	"plus",
	"comma",
	"hyphen",
	"period",
	"slash",
	"zero",
	"one",
	"two",
	"three",
	"four",
	"five",
	"six",
	"seven",
	"eight",
	"nine",
	"colon",
	"semicolon",
	"less",
	"equal",
	"greater",
	"question",
	"at",
	"A",
	"B",
	"C",
	"D",
	"E",
	"F",
	"G",
	"H",
	"I",
	"J",
	"K",
	"L",
	"M",
	"N",
	"O",
	"P",
	"Q",
	"R",
	"S",
	"T",
	"U",
	"V",
	"W",
	"X",
	"Y",
	"Z",
	"bracketleft",
	"backslash",
	"bracketright",
	"asciicircum",
	"underscore",
	"quoteleft",
	"a",
	"b",
	"c",
	"d",
	"e",
	"f",
	"g",
	"h",
	"i",
	"j",
	"k",
	"l",
	"m",
	"n",
	"o",
	"p",
	"q",
	"r",
	"s",
	"t",
	"u",
	"v",
	"w",
	"x",
	"y",
	"z",
	"braceleft",
	"bar",
	"braceright",
	"asciitilde",
	"exclamdown",
	"cent",
	"sterling",
	"fraction",
	"yen",
	"florin",
	"section",
	"currency",
	"quotesingle",
	"quotedblleft",
	"guillemotleft",
	"guilsinglleft",
	"guilsinglright",
	"fi",
	"fl",
	"endash",
	"dagger",
	"daggerdbl",
	"periodcentered",
	"paragraph",
	"bullet",
	"quotesinglbase",
	"quotedblbase",
	"quotedblright",
	"guillemotright",
	"ellipsis",
	"perthousand",
	"questiondown",
	"grave",
	"acute",
	"circumflex",
	"tilde",
	"macron",
	"breve",
	"dotaccent",
	"dieresis",
	"ring",
	"cedilla",
	"hungarumlaut",
	"ogonek",
	"caron",
	"emdash",
	"AE",
	"ordfeminine",
	"Lslash",
	"Oslash",
	"OE",
	"ordmasculine",
	"ae",
	"dotlessi",
	"lslash",
	"oslash",
	"oe",
	"germandbls",
	"onesuperior",
	"logicalnot",
	"mu",
	"trademark",
	"Eth",
	"onehalf",
	"plusminus",
	"Thorn",
	"onequarter",
	"divide",
	"brokenbar",
	"degree",
	"thorn",
	"threequarters",
	"twosuperior",
	"registered",
	"minus",
	"eth",
	"multiply",
	"threesuperior",
	"copyright",
	"Aacute",
	"Acircumflex",
	"Adieresis",
	"Agrave",
	"Aring",
	"Atilde",
	"Ccedilla",
	"Eacute",
	"Ecircumflex",
	"Edieresis",
	"Egrave",
	"Iacute",
	"Icircumflex",
	"Idieresis",
	"Igrave",
	"Ntilde",
	"Oacute",
	"Ocircumflex",
	"Odieresis",
	"Ograve",
	"Otilde",
	"Scaron",
	"Uacute",
	"Ucircumflex",
	"Udieresis",
	"Ugrave",
	"Yacute",
	"Ydieresis",
	"Zcaron",
	"aacute",
	"acircumflex",
	"adieresis",
	"agrave",
	"aring",
	"atilde",
	"ccedilla",
	"eacute",
	"ecircumflex",
	"edieresis",
	"egrave",
	"iacute",
	"icircumflex",
	"idieresis",
	"igrave",
	"ntilde",
	"oacute",
	"ocircumflex",
	"odieresis",
	"ograve",
	"otilde",
	"scaron",
	"uacute",
	"ucircumflex",
	"udieresis",
	"ugrave",
	"yacute",
	"ydieresis",
	"zcaron",
	"exclamsmall",
	"Hungarumlautsmall",
	"dollaroldstyle",
	"dollarsuperior",
	"ampersandsmall",
	"Acutesmall",
	"parenleftsuperior",
	"parenrightsuperior",
	"twodotenleader",
	"onedotenleader",
	"zerooldstyle",
	"oneoldstyle",
	"twooldstyle",
	"threeoldstyle",
	"fouroldstyle",
	"fiveoldstyle",
	"sixoldstyle",
	"sevenoldstyle",
	"eightoldstyle",
	"nineoldstyle",
	"commasuperior",
	"threequartersemdash",
	"periodsuperior",
	"questionsmall",
	"asuperior",
	"bsuperior",
	"centsuperior",
	"dsuperior",
	"esuperior",
	"isuperior",
	"lsuperior",
	"msuperior",
	"nsuperior",
	"osuperior",
	"rsuperior",
	"ssuperior",
	"tsuperior",
	"ff",
	"ffi",
	"ffl",
	"parenleftinferior",
	"parenrightinferior",
	"Circumflexsmall",
	"hyphensuperior",
	"Gravesmall",
	"Asmall",
	"Bsmall",
	"Csmall",
	"Dsmall",
	"Esmall",
	"Fsmall",
	"Gsmall",
	"Hsmall",
	"Ismall",
	"Jsmall",
	"Ksmall",
	"Lsmall",
	"Msmall",
	"Nsmall",
	"Osmall",
	"Psmall",
	"Qsmall",
	"Rsmall",
	"Ssmall",
	"Tsmall",
	"Usmall",
	"Vsmall",
	"Wsmall",
	"Xsmall",
	"Ysmall",
	"Zsmall",
	"colonmonetary",
	"onefitted",
	"rupiah",
	"Tildesmall",
	"exclamdownsmall",
	"centoldstyle",
	"Lslashsmall",
	"Scaronsmall",
	"Zcaronsmall",
	"Dieresissmall",
	"Brevesmall",
	"Caronsmall",
	"Dotaccentsmall",
	"Macronsmall",
	"figuredash",
	"hypheninferior",
	"Ogoneksmall",
	"Ringsmall",
	"Cedillasmall",
	"questiondownsmall",
	"oneeighth",
	"threeeighths",
	"fiveeighths",
	"seveneighths",
	"onethird",
	"twothirds",
	"zerosuperior",
	"foursuperior",
	"fivesuperior",
	"sixsuperior",
	"sevensuperior",
	"eightsuperior",
	"ninesuperior",
	"zeroinferior",
	"oneinferior",
	"twoinferior",
	"threeinferior",
	"fourinferior",
	"fiveinferior",
	"sixinferior",
	"seveninferior",
	"eightinferior",
	"nineinferior",
	"centinferior",
	"dollarinferior",
	"periodinferior",
	"commainferior",
	"Agravesmall",
	"Aacutesmall",
	"Acircumflexsmall",
	"Atildesmall",
	"Adieresissmall",
	"Aringsmall",
	"AEsmall",
	"Ccedillasmall",
	"Egravesmall",
	"Eacutesmall",
	"Ecircumflexsmall",
	"Edieresissmall",
	"Igravesmall",
	"Iacutesmall",
	"Icircumflexsmall",
	"Idieresissmall",
	"Ethsmall",
	"Ntildesmall",
	"Ogravesmall",
	"Oacutesmall",
	"Ocircumflexsmall",
	"Otildesmall",
	"Odieresissmall",
	"OEsmall",
	"Oslashsmall",
	"Ugravesmall",
	"Uacutesmall",
	"Ucircumflexsmall",
	"Udieresissmall",
	"Yacutesmall",
	"Thornsmall",
	"Ydieresissmall",
	"001.000",
	"001.001",
	"001.002",
	"001.003",
	"Black",
	"Bold",
	"Book",
	"Light",
	"Medium",
	"Regular",
	"Roman",
	"Semibold"}
