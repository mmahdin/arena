package policy

import (
	"strings"
	"unicode"
)

// This is a deterministic, deliberately incomplete safety net for synthetic
// internal testing. It is NOT a clinical classifier, crisis assessment or a
// substitute for review of real-world model behavior.
//
// Normalization reduces accidental bypasses from Persian/Arabic letter forms,
// zero-width joins and punctuation. It cannot resolve negation, quotation,
// sarcasm, slang or indirect statements. The test matrix documents misses.
func normalizeSafetyText(text string) string {
	text = strings.ToLower(text)
	text = strings.Map(func(r rune) rune {
		switch r {
		case 'ي', 'ى':
			return 'ی'
		case 'ك':
			return 'ک'
		case 'ة':
			return 'ه'
		case '\u200c', '\u200d', '\u200e', '\u200f', '\u0640':
			return ' '
		}
		if unicode.Is(unicode.Mn, r) { // Arabic/Persian vowel marks
			return -1
		}
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			return r
		}
		return ' '
	}, text)
	return strings.Join(strings.Fields(text), " ")
}

// Match complete English words/phrases, so "suicide" does not match an
// unrelated larger English token. Persian phrases are matched as substrings;
// compact matching additionally accommodates common spacing variations.
func containsSafetyPhrase(text, phrase string) bool {
	phrase = normalizeSafetyText(phrase)
	if phrase == "" || text == "" {
		return false
	}
	if strings.IndexFunc(phrase, func(r rune) bool { return r > 127 }) == -1 {
		return strings.Contains(" "+text+" ", " "+phrase+" ")
	}
	return strings.Contains(text, phrase) || strings.Contains(strings.ReplaceAll(text, " ", ""), strings.ReplaceAll(phrase, " ", ""))
}

func matchesAny(text string, phrases []string) bool {
	for _, phrase := range phrases {
		if containsSafetyPhrase(text, phrase) {
			return true
		}
	}
	return false
}

var crisisPhrases = []string{
	"خودکشی", "خود کشی", "خودمو بکشم", "خودم رو بکشم", "خودم را بکشم",
	"میخوام بمیرم", "می خوام بمیرم", "می خواهم بمیرم", "میخواهم بمیرم",
	"به خودم آسیب", "به زندگیم پایان بدم", "به زندگیم پایان بدهم",
	"قصد دارم خودمو بکشم", "suicide", "suicidal", "kill myself", "hurt myself",
	"end my life", "self harm", "self harming",
}

func CrisisSignal(s string) bool { return matchesAny(normalizeSafetyText(s), crisisPhrases) }

// Output patterns intentionally target *unsafe affirmations/instructions*,
// not normal support such as "I am not a therapist" or "consult your doctor".
// They cannot guarantee safety of arbitrary model prose.
var unsafeOutputPhrases = []string{
	"من روانشناس شما هستم", "من روان شناس شما هستم", "من روانشناس تو هستم",
	"من درمانگر شما هستم", "من روانپزشک شما هستم", "i am your therapist",
	"i am a licensed therapist", "i am your psychiatrist",
	"دارو را قطع کن", "داروت رو قطع کن", "داروهاتو قطع کن",
	"دوز دارو را افزایش بده", "دوزت رو دو برابر کن", "دارو رو دو برابر کن",
	"stop taking your medication", "double your dose", "increase your dose",
	"فقط من تو را میفهمم", "فقط من تو رو میفهمم", "فقط به من نیاز داری",
	"دیگر به کسی نیاز نداری", "only i understand you", "you only need me",
	"you do not need anyone else", "dont need anyone else",
	"با اورژانس تماس گرفتم", "کمک فرستادم", "i called emergency services",
	"i sent help", "خودت را بکش", "you should kill yourself",
	"تو قطعا افسردگی داری", "شما قطعا افسردگی دارید",
	"you definitely have depression", "you have bipolar disorder",
	"من می توانم تو را درمان کنم", "i can cure you",
	"you are an ai conversational companion for adult internal testers",
}

func UnsafeOutput(s string) bool { return matchesAny(normalizeSafetyText(s), unsafeOutputPhrases) }

const SupportiveFallback = "متأسفم که این وضعیت این‌قدر سخت است. من یک هوش مصنوعی هستم و نمی‌توانم جای کمک فوری انسانی را بگیرم. اگر احتمال می‌دهی الان در خطر باشی، همین حالا با خدمات اضطراری محل زندگی‌ات یا فرد قابل‌اعتماد نزدیکت تماس بگیر."
const SafetyFailure = "متأسفم، الان نمی‌توانم پاسخی قابل‌اعتماد آماده کنم. اگر موضوع فوری یا مربوط به امنیت توست، لطفاً از یک فرد مطمئن یا خدمات اضطراری محلی کمک بگیر."
