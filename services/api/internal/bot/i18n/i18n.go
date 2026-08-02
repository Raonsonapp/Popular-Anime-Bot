// Package i18n holds the bot's UI translation catalog. It only covers
// fixed interface text (menus, labels, status messages) - anime content
// (titles, genres, synopsis) comes from the catalog in whatever language
// the MTProto listener translated it into and isn't re-translated here.
package i18n

import "fmt"

type Lang string

const (
	Tajik   Lang = "tg"
	Persian Lang = "fa"
	Russian Lang = "ru"
	English Lang = "en"
)

// Default is used for brand-new users and as the fallback when a stored
// value doesn't match a known language.
const Default = Tajik

// Normalize maps an arbitrary stored/user-supplied string to a known Lang,
// falling back to Default.
func Normalize(s string) Lang {
	switch Lang(s) {
	case Tajik, Persian, Russian, English:
		return Lang(s)
	default:
		return Default
	}
}

// All lists the supported languages in menu display order.
var All = []Lang{Tajik, Persian, Russian, English}

// DisplayName returns the label shown for this language in the language
// picker (in its own language, with a flag).
func (l Lang) DisplayName() string {
	switch l {
	case Persian:
		return "🇮🇷 فارسی"
	case Russian:
		return "🇷🇺 Русский"
	case English:
		return "🇬🇧 English"
	default:
		return "🇹🇯 Тоҷикӣ"
	}
}

// T looks up a translated string for key in lang, falling back to Default
// and finally to the raw key if nothing matches.
func T(lang Lang, key string) string {
	entry, ok := catalog[key]
	if !ok {
		return key
	}
	if v, ok := entry[lang]; ok {
		return v
	}
	if v, ok := entry[Default]; ok {
		return v
	}
	return key
}

// Tf is T plus fmt.Sprintf formatting for keys with placeholders.
func Tf(lang Lang, key string, args ...interface{}) string {
	return fmt.Sprintf(T(lang, key), args...)
}

var catalog = map[string]map[Lang]string{
	"menu_search": {
		Tajik: "🔍 Ҷустуҷӯ", Persian: "🔍 جستجو", Russian: "🔍 Поиск", English: "🔍 Search",
	},
	"menu_genres": {
		Tajik: "🎭 Жанрҳо", Persian: "🎭 ژانرها", Russian: "🎭 Жанры", English: "🎭 Genres",
	},
	"menu_top_rated": {
		Tajik: "⭐ Беҳтаринҳо", Persian: "⭐ برترین‌ها", Russian: "⭐ Топ рейтинга", English: "⭐ Top Rated",
	},
	"menu_trending": {
		Tajik: "🔥 Маъмултарин", Persian: "🔥 پرطرفدار", Russian: "🔥 В тренде", English: "🔥 Trending",
	},
	"menu_newest": {
		Tajik: "🆕 Навтарин", Persian: "🆕 جدیدترین", Russian: "🆕 Новинки", English: "🆕 Newest",
	},
	"menu_random": {
		Tajik: "🎲 Тасодуфӣ", Persian: "🎲 تصادفی", Russian: "🎲 Случайное", English: "🎲 Random",
	},
	"menu_continue": {
		Tajik: "▶️ Идомаи тамошо", Persian: "▶️ ادامه تماشا", Russian: "▶️ Продолжить просмотр", English: "▶️ Continue Watching",
	},
	"menu_favorites": {
		Tajik: "❤️ Дӯстдоштаҳо", Persian: "❤️ علاقه‌مندی‌ها", Russian: "❤️ Избранное", English: "❤️ Favorites",
	},
	"menu_history": {
		Tajik: "🕒 Таърих", Persian: "🕒 تاریخچه", Russian: "🕒 История", English: "🕒 History",
	},
	"menu_language": {
		Tajik: "🌐 Забон", Persian: "🌐 زبان", Russian: "🌐 Язык", English: "🌐 Language",
	},
	"back_to_menu": {
		Tajik: "🔙 Бозгашт ба меню", Persian: "🔙 بازگشت به منو", Russian: "🔙 Назад в меню", English: "🔙 Back to menu",
	},
	"back_to_anime": {
		Tajik: "🔙 Бозгашт ба аниме", Persian: "🔙 بازگشت به آنیمه", Russian: "🔙 Назад к аниме", English: "🔙 Back to anime",
	},
	"prev": {
		Tajik: "⬅️ Қаблӣ", Persian: "⬅️ قبلی", Russian: "⬅️ Назад", English: "⬅️ Prev",
	},
	"next": {
		Tajik: "➡️ Баъдӣ", Persian: "➡️ بعدی", Russian: "➡️ Далее", English: "➡️ Next",
	},
	"watch": {
		Tajik: "▶️ Тамошо", Persian: "▶️ تماشا", Russian: "▶️ Смотреть", English: "▶️ Watch",
	},
	"add_favorite": {
		Tajik: "⭐ Илова ба дӯстдоштаҳо", Persian: "⭐ افزودن به علاقه‌مندی", Russian: "⭐ В избранное", English: "⭐ Add to favorites",
	},
	"remove_favorite": {
		Tajik: "💔 Хориҷ аз дӯстдоштаҳо", Persian: "💔 حذف از علاقه‌مندی", Russian: "💔 Убрать из избранного", English: "💔 Remove favorite",
	},
	"share": {
		Tajik: "📢 Нашр кардан", Persian: "📢 اشتراک‌گذاری", Russian: "📢 Поделиться", English: "📢 Share",
	},

	"welcome": {
		Tajik:   "👋 Хуш омадед ба <b>AnimeBot</b>!\n\nБеҳтарин анимеҳоро дар ин ҷо тамошо кунед 🎬\n\nБарои ҷустуҷӯ, номи анимеро нависед ё аз менюи зер истифода баред.",
		Persian: "👋 به <b>AnimeBot</b> خوش آمدید!\n\nبهترین انیمه‌ها را اینجا تماشا کنید 🎬\n\nبرای جستجو، نام انیمه را تایپ کنید یا از منوی زیر استفاده کنید.",
		Russian: "👋 Добро пожаловать в <b>AnimeBot</b>!\n\nЛучшие аниме смотрите прямо здесь 🎬\n\nЧтобы найти аниме, просто напишите его название или воспользуйтесь меню ниже.",
		English: "👋 Welcome to <b>AnimeBot</b>!\n\nWatch the best anime right here 🎬\n\nTo search, just type an anime's name or use the menu below.",
	},
	"search_prompt": {
		Tajik: "🔍 Номи анимеи мавриди назарро нависед:", Persian: "🔍 نام انیمه مورد نظر خود را تایپ کنید:",
		Russian: "🔍 Напишите название аниме:", English: "🔍 Type the anime's name:",
	},
	"search_results_header": {
		Tajik: "🔎 Натиҷаҳои ҷустуҷӯ барои: %s", Persian: "🔎 نتایج جستجو برای: %s",
		Russian: "🔎 Результаты поиска: %s", English: "🔎 Search results for: %s",
	},
	"search_error": {
		Tajik: "❌ Хатогӣ ҳангоми ҷустуҷӯ. Лутфан дубора кӯшиш кунед.", Persian: "❌ خطا در جستجو. لطفاً دوباره تلاش کنید.",
		Russian: "❌ Ошибка поиска. Попробуйте ещё раз.", English: "❌ Search error. Please try again.",
	},
	"nothing_found": {
		Tajik: "Чизе ёфт нашуд. 🔎", Persian: "چیزی پیدا نشد. 🔎",
		Russian: "Ничего не найдено. 🔎", English: "Nothing found. 🔎",
	},
	"anime_not_found": {
		Tajik: "❌ Мутаассифона, ин аниме ёфт нашуд.", Persian: "❌ متأسفانه این انیمه یافت نشد.",
		Russian: "❌ К сожалению, это аниме не найдено.", English: "❌ Sorry, this anime wasn't found.",
	},
	"episodes_header": {
		Tajik: "🎞 Қисмҳо (саҳифаи %d):", Persian: "🎞 قسمت‌ها (صفحه %d):",
		Russian: "🎞 Серии (страница %d):", English: "🎞 Episodes (page %d):",
	},
	"no_episodes": {
		Tajik: "Ҳанӯз ягон қисм барои ин аниме мунташир нашудааст.", Persian: "هنوز قسمتی برای این انیمه منتشر نشده است.",
		Russian: "Для этого аниме пока нет серий.", English: "No episodes have been published for this anime yet.",
	},
	"episodes_error": {
		Tajik: "❌ Хатогӣ ҳангоми гирифтани рӯйхати қисмҳо.", Persian: "❌ خطا در دریافت لیست قسمت‌ها.",
		Russian: "❌ Ошибка при получении списка серий.", English: "❌ Failed to load the episode list.",
	},
	"episode_not_found": {
		Tajik: "❌ Ин қисм ёфт нашуд.", Persian: "❌ این قسمت یافت نشد.",
		Russian: "❌ Эта серия не найдена.", English: "❌ This episode wasn't found.",
	},
	"deliver_error": {
		Tajik:   "❌ Ҳангоми фиристодани файл хатогӣ рӯй дод. Лутфан баъдтар дубора кӯшиш кунед.",
		Persian: "❌ در ارسال فایل خطایی رخ داد. لطفاً بعداً دوباره تلاش کنید.",
		Russian: "❌ Ошибка при отправке файла. Попробуйте позже.",
		English: "❌ Failed to send the file. Please try again later.",
	},
	"favorite_added": {
		Tajik: "⭐ Ба дӯстдоштаҳо илова шуд.", Persian: "⭐ به علاقه‌مندی‌ها اضافه شد.",
		Russian: "⭐ Добавлено в избранное.", English: "⭐ Added to favorites.",
	},
	"favorite_removed": {
		Tajik: "💔 Аз дӯстдоштаҳо хориҷ шуд.", Persian: "💔 از علاقه‌مندی‌ها حذف شد.",
		Russian: "💔 Убрано из избранного.", English: "💔 Removed from favorites.",
	},
	"favorite_error": {
		Tajik: "❌ Хатогӣ ҳангоми сабти дӯстдошта.", Persian: "❌ خطا در ثبت علاقه‌مندی.",
		Russian: "❌ Ошибка при сохранении избранного.", English: "❌ Failed to update favorites.",
	},
	"favorites_empty": {
		Tajik: "Шумо ҳанӯз чизе ба дӯстдоштаҳо илова накардаед. ❤️", Persian: "هنوز چیزی به علاقه‌مندی‌ها اضافه نکرده‌اید. ❤️",
		Russian: "Вы пока ничего не добавили в избранное. ❤️", English: "You haven't added anything to favorites yet. ❤️",
	},
	"favorites_header": {
		Tajik: "❤️ Дӯстдоштаҳои шумо:", Persian: "❤️ علاقه‌مندی‌های شما:",
		Russian: "❤️ Ваше избранное:", English: "❤️ Your favorites:",
	},
	"history_error": {
		Tajik: "❌ Хатогӣ ҳангоми гирифтани таърих.", Persian: "❌ خطا در دریافت تاریخچه.",
		Russian: "❌ Ошибка при получении истории.", English: "❌ Failed to load history.",
	},
	"history_empty": {
		Tajik: "Таърихе мавҷуд нест. 🕒", Persian: "تاریخچه‌ای موجود نیست. 🕒",
		Russian: "История пуста. 🕒", English: "No history yet. 🕒",
	},
	"history_header": {
		Tajik: "🕒 Таърихи тамошои шумо:", Persian: "🕒 تاریخچه تماشای شما:",
		Russian: "🕒 История ваших просмотров:", English: "🕒 Your watch history:",
	},
	"history_episode_label": {
		Tajik: "Қисми #%d", Persian: "قسمت #%d",
		Russian: "Серия #%d", English: "Episode #%d",
	},
	"genres_prompt": {
		Tajik: "🎭 Як жанрро интихоб кунед:", Persian: "🎭 یک ژانر را انتخاب کنید:",
		Russian: "🎭 Выберите жанр:", English: "🎭 Pick a genre:",
	},
	"genres_error": {
		Tajik: "❌ Хатогӣ ҳангоми гирифтани жанрҳо.", Persian: "❌ خطا در دریافت ژانرها.",
		Russian: "❌ Ошибка при получении жанров.", English: "❌ Failed to load genres.",
	},
	"random_not_found": {
		Tajik: "❌ Ягон аниме ёфт нашуд.", Persian: "❌ آنیمه‌ای یافت نشد.",
		Russian: "❌ Аниме не найдено.", English: "❌ No anime found.",
	},
	"generic_error": {
		Tajik: "❌ Хатогӣ.", Persian: "❌ خطا.",
		Russian: "❌ Ошибка.", English: "❌ Error.",
	},
	"list_error": {
		Tajik: "❌ Хатогӣ ҳангоми гирифтани рӯйхат.", Persian: "❌ خطا در دریافت لیست.",
		Russian: "❌ Ошибка при получении списка.", English: "❌ Failed to load the list.",
	},

	"choose_language": {
		Tajik: "🌐 Забони худро интихоб кунед:", Persian: "🌐 زبان خود را انتخاب کنید:",
		Russian: "🌐 Выберите язык:", English: "🌐 Choose your language:",
	},
	"language_changed": {
		Tajik: "✅ Забон ба тоҷикӣ иваз шуд.", Persian: "✅ زبان به فارسی تغییر یافت.",
		Russian: "✅ Язык изменён на русский.", English: "✅ Language changed to English.",
	},

	"top_rated_title": {
		Tajik: "⭐ Беҳтарин анимеҳо", Persian: "⭐ برترین انیمه‌ها",
		Russian: "⭐ Лучшие аниме", English: "⭐ Top rated anime",
	},
	"trending_title": {
		Tajik: "🔥 Маъмултаринҳо", Persian: "🔥 پرطرفدارترین‌ها",
		Russian: "🔥 В тренде", English: "🔥 Trending now",
	},
	"newest_title": {
		Tajik: "🆕 Навтаринҳо", Persian: "🆕 جدیدترین‌ها",
		Russian: "🆕 Новинки", English: "🆕 Newest releases",
	},
	"genre_results_title": {
		Tajik: "🎭 Анимеҳои ин жанр:", Persian: "🎭 انیمه‌های این ژانر:",
		Russian: "🎭 Аниме этого жанра:", English: "🎭 Anime in this genre:",
	},

	"caption_episodes_suffix": {
		Tajik: "(%d қисм)", Persian: "(%d قسمت)",
		Russian: "(%d серий)", English: "(%d episodes)",
	},
	"episode_button": {
		Tajik: "Қисми %d [%s]", Persian: "قسمت %d [%s]",
		Russian: "Серия %d [%s]", English: "Episode %d [%s]",
	},

	"caption_genre": {
		Tajik: "🎭 Жанр: %s", Persian: "🎭 ژانر: %s",
		Russian: "🎭 Жанр: %s", English: "🎭 Genre: %s",
	},
	"caption_studio": {
		Tajik: "🏢 Студия: %s", Persian: "🏢 استودیو: %s",
		Russian: "🏢 Студия: %s", English: "🏢 Studio: %s",
	},
	"caption_year": {
		Tajik: "📅 Сол: %d", Persian: "📅 سال: %d",
		Russian: "📅 Год: %d", English: "📅 Year: %d",
	},
	"caption_status": {
		Tajik: "📺 Ҳолат: %s", Persian: "📺 وضعیت: %s",
		Russian: "📺 Статус: %s", English: "📺 Status: %s",
	},
	"caption_rating": {
		Tajik: "⭐ Баҳо: %.1f", Persian: "⭐ امتیاز: %.1f",
		Russian: "⭐ Рейтинг: %.1f", English: "⭐ Rating: %.1f",
	},
	"caption_episodes_count": {
		Tajik: "🎞 Шумораи қисмҳо: %d", Persian: "🎞 تعداد قسمت‌ها: %d",
		Russian: "🎞 Количество серий: %d", English: "🎞 Episode count: %d",
	},
	"caption_duration": {
		Tajik: "⏱ Давомнокии ҳар қисм: %d дақиқа", Persian: "⏱ مدت هر قسمت: %d دقیقه",
		Russian: "⏱ Длительность серии: %d мин.", English: "⏱ Duration per episode: %d min",
	},
	"caption_synopsis": {
		Tajik: "📝 Мухтасари сюжет:\n%s", Persian: "📝 خلاصه داستان:\n%s",
		Russian: "📝 Краткое описание:\n%s", English: "📝 Synopsis:\n%s",
	},

	"status_ongoing": {
		Tajik: "🟢 Дар ҳоли пахш", Persian: "🟢 در حال پخش",
		Russian: "🟢 Онгоинг", English: "🟢 Ongoing",
	},
	"status_completed": {
		Tajik: "✅ Анҷомёфта", Persian: "✅ تکمیل‌شده",
		Russian: "✅ Завершено", English: "✅ Completed",
	},
	"status_announced": {
		Tajik: "📢 Эълоншуда", Persian: "📢 اعلام‌شده",
		Russian: "📢 Анонсировано", English: "📢 Announced",
	},
}
