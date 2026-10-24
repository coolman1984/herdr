# herdr على الموبايل (أندرويد + Termux)

موبايلك يبقى جهاز برمجة كامل: herdr + كل وكلاء البرمجة + Hermes Agent، وترفع على GitHub من أي مكان.

## الفكرة في سطرين
- Termux بيشغّل جواه "ورشة" Ubuntu صغيرة (proot)، لأن herdr ومعظم الأدوات مبنية للينكس العادي.
- جوه الورشة: herdr، git، gh، Node.js، Claude Code، Codex، Gemini، OpenCode، Copilot، Hermes Agent — وكلهم بيتحدّثوا لوحدهم كل 12 ساعة.

## التركيب (مرة واحدة)
1. نزّل **Termux** و **Termux:API** و **Termux:Boot** من F-Droid (مش Google Play).
2. افتح Termux والصق:

```bash
curl -fsSL https://raw.githubusercontent.com/coolman1984/herdr/main/mobile/termux/setup.sh | bash
```

3. هيسألك عن اسمك وإيميلك، ويديك كود تسجيل دخول GitHub تفتحه في المتصفح.
4. من إعدادات أندرويد: Termux ← البطارية ← **بدون قيود**. وافتح Termux:Boot مرة واحدة.

## الاستخدام اليومي
| تكتب | يحصل |
| --- | --- |
| `dev` | يفتح herdr في `~/code` |
| `dev shell` | سطر أوامر عادي جوه الورشة |
| `dev update` | يحدّث كل حاجة دلوقتي |
| `radar` | يكشف المشاكل، يصلّحها، يبعت تقرير لـ Claude، ويفتح herdr |
| `radar hermes` | يعيد تركيب Hermes من الصفر (إعداداتك ومفاتيحك محفوظة) |
| `ship "رسالة"` | يحفظ كل التعديلات ويرفعها على GitHub (بيعيد المحاولة لو النت فصل) |

- ملفات الموبايل موجودة جوه الورشة في `/sdcard`.
- تختار الأدوات من `~/.workbench/tools.conf` (حط `0` قدام أي أداة مش عاوزها).
- Hermes: أول مرة اكتب `hermes setup` تختار المزوّد والمفتاح.

## الرادار (لو حاجة باظت)
- أول مرة: `curl -fsSL https://raw.githubusercontent.com/coolman1984/herdr/main/mobile/termux/radar.sh | bash`
- بعد كده اكتب `radar` بس. بيحدّث نفسه لوحده كل مرة.
- التقرير بيروح لمستودع خاص عندك اسمه `phone-radar` (المفاتيح والتوكنات بتتشال منه)، وClaude يقراه ويصلّح الأداة، وإنت تشغّل `radar` تاني.

## حدود لازم تعرفها
- أندرويد ممكن يقفل Termux في الخلفية؛ "بدون قيود" للبطارية + `termux-wake-lock` بيقللوا ده.
- الورشة أبطأ شوية من الكمبيوتر (proot)، بس كفاية جدًا للبرمجة والوكلاء.
- باكدج Hermes الرسمي لـ Termux معطّل حاليًا حسب توثيقهم، فبنركّبه جوه الورشة بالطريقة الرسمية للينكس.
