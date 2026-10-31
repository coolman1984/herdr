# رد على المراجعة المستقلة الثانية (PR #17) — ٣١ أكتوبر ٢٠٢٦

النسخة المراجَعة: `cd7b35e`. خط الأساس قبل أي إصلاح: `docs/audits/round2-baseline.md` على `13b468b`
(فشل الاختبارات الثلاثة عشر كلها: ٦ في المحرك و٧ في مسار المستندات).
لكل بند: إعادة الإنتاج ← القاعدة المنتهكة ← الإصلاح (المرحلة في HISTORY) ← الاختبار ← ما بقي.
اختبارات المراجعة الأصلية لم تُعدَّل ولم تُتخطَّ. تغيّر شكل اختبارات كتبتُها أنا سابقًا فقط حين تغيّر السلوك
المطلوب، ومذكور ذلك في HISTORY (مثل الإجماليات التي كانت تحسب صف المجموع مرتين).

## البنود الثلاثة عشر

| بند | إعادة الإنتاج | القاعدة | الإصلاح | الاختبار |
|---|---|---|---|---|
| D1 P0 | قراءة البصمة قبل التوجيه رفعت PermissionError | لا قراءة لمحتوى ملف شركة قبل السياسة | التوجيه من الاسم فقط؛ هوية بالبيانات الوصفية (65) | `test_company_authorized_route_does_not_open_source_bytes` + `test_no_company_file_of_any_type_is_read_before_policy` (٨ أنواع) |
| D2 P0 | CSV شركة → native | كل ملفات الشركة عبر التطبيق المصرح | CSV/TSV/TXT/PRN → إكسيل COM؛ PDF وغير المعروف → قدرة غير متاحة موثقة (65)؛ الملف «unknown» صارم (66) | `test_company_csv_requires_authorized_application`، `test_unknown_device_profile_is_strict` |
| D3 | عمودان باسم Amount → قيمة واحدة | لا ضياع صامت | أسماء فريدة + ملاحظة + مطابقة أعداد (65) | `test_duplicate_headers_never_silently_discard_values`، `test_headers_duplicate_missing_multiline_reconciled` |
| D4 | `hello\nworld` → `helloworld` | النص الأصلي كما هو | قراءة متدفقة بـ `newline=""` (65) | `test_quoted_csv_newline_is_preserved` |
| D5 | تقريب إلى 1.23E+29 | مجاميع دقيقة | دقة 200 رقم واعتراض Inexact (65) | `test_large_decimal_total_keeps_cents`، `test_inexact_total_is_an_error_not_a_rounding` |
| D6 | 1904 → 1899-12-31 | نظام التاريخ من المصنف | Date1904 من المصنف ومن عامل COM؛ خطأ 1900-02-29 (65) | `test_com_1904_workbook_date_system_is_respected`، `test_date_systems_leap_boundary_and_time`، `test_workbook_1904_flag_is_read` |
| D7 | خلية `t="d"` اختفت | لا قيمة تختفي | قراءة ISO + احتفاظ بأي نوع مجهول نصًا (65) | `test_iso_date_cell_is_not_accepted_as_empty`، `test_unknown_cell_kind_is_kept` |
| G1 | 99% قديمة بلا موعد → مؤهل | المجهول ليس صفرًا | العمر وحده لا يعيد فتح حساب مستنفد (64) | `TestIndependentRound2StaleExhaustionWithoutResetIsNotFree` |
| G2 | Left=0 → مؤهل | نفاد الرصيد يوقف | Left<=0 يمنع (64) | `TestIndependentRound2RemainingZeroIsNotFree` |
| G3 | نافذة بلا قيمة → ثقة كاملة | المجهول ثقة منخفضة | (64) | `TestIndependentRound2MissingUsageIsLowTrust` |
| G4 | ميزانية صفر تُفسَّر بلا حد | احترام صفر وكلاء | 0 = صفر، السالب = بلا حد، لا رفع إلى 1 (64) | `TestIndependentRound2ZeroAgentBudgetDefersWork` |
| G5 P0 | نتيجة تُفحص والكاتب حي | لا انتقال قبل تأكيد التوقف | كل مسارات التحرير تنتظر التأكيد (64) | `TestIndependentRound2ResultWaitsForConfirmedStop`، `TestStuckStopBlocksEveryReleasePath`، `TestResultIsHandledOnlyAfterTheStopIsConfirmed` |
| G6 P0 | كاتبان لمهمة واحدة | لا تداخل منفذين | التبديل بعد التأكيد فقط + نقطة حفظ (64) | `TestIndependentRound2QuotaRotationWaitsForConfirmedStop` |

## الملاحظات الإضافية الست عشرة

| # | الحالة | ما تغيّر | الدليل |
|---|---|---|---|
| 1 | أُصلحت | الوظيفة الملغاة تبقى مملوكة ومحسوبة حتى تنتهي؛ لا كاتب فوق وظيفة قديمة | `TestCancelledJobStaysOwnedUntilJoined` |
| 2 | أُصلحت | توقيع HMAC بدل الرمز؛ مفاتيح في keys/ بصلاحية 0600؛ صندوق 0600؛ رفض التكرار؛ لا مفتاح في أي ملف آخر. **باقٍ:** عملية بنفس حساب المستخدم تقرأ keys/ — العزل الحقيقي حساب نظام منفصل | `TestEnqueueNeverWritesTheKey`، `TestInboxRoleComesFromSignatureNotClaim` |
| 3 | جزئيًا | مسارات محمية لا تدمج؛ المراجع في نسخة منفصلة للإيداع المراجَع. **باقٍ:** لا عزل نظام تشغيل للكاتب عن ملفات الهدف | `TestProtectedFilesAndReviewerSnapshot` |
| 4 | جزئيًا | عامل COM يسجل عملية أوفيس التي أنشأها ولا يغلق نسخة المستخدم؛ القتل عند المهلة لعمليته فقط. **غير مثبت:** على أوفيس حقيقي — بوابات g3/g4 في `tools/device-harness` | تحليل ٥٫١/٧ في CI؛ بوابة الجهاز |
| 5 | أُصلحت (غير مثبتة على ناسكا) | مجلد عمل 0700 يُحذف عند النجاح والفشل؛ لا تصدير قيم دون موافقة؛ قاعدة 0600 | `test_com_output_is_removed_even_when_the_worker_fails`، `test_company_run_without_approval_exports_no_values` |
| 6 | أُصلحت | لا `-ExecutionPolicy Bypass`؛ السياسة المانعة قدرة موثقة | `test_blocked_execution_policy_is_reported_not_bypassed` |
| 7 | أُصلحت جزئيًا | CSV متدفق (٥٠٠ ألف صف = 25MB)، عامل COM يقرأ دفعات ويكتب JSONL، تصدير متدفق، مصادر محدودة. **غير مثبت:** ذاكرة COM على أوفيس حقيقي | قياس محلي + `test_csv_export_neutralises_formulas_and_streams` |
| 8 | أُصلحت جزئيًا | حدود قبل التخصيص (الدمج، السلاسل المشتركة، حجم XML، الصفوف) | `test_huge_merge_is_not_expanded_and_shared_strings_are_bounded` |
| 9 | أُصلحت | هوية تشمل السياسة وإصدار المحلل؛ قفل استيراد؛ نقطة حفظ لا ترجع؛ ترحيل يعيد القراءة داخل القفل؛ تغير المصدر؛ امتلاء القرص | `test_second_importer_waits_for_the_lease`، `test_checkpoint_never_moves_back`، `test_concurrent_migration_applies_once`، `test_source_change_during_import_keeps_nothing`، `test_disk_full_is_a_clear_status_and_the_database_stays_sound` |
| 10 | أُصلحت جزئيًا | العملة من تنسيق الخلية؛ تحذير ذاكرة المعادلات القديمة؛ هوية جداول وورد/باوربوينت؛ خلايا المعادلات (المجاميع) لا تُجمع مرتين | `test_formula_total_row_is_not_added_again` |
| 11 | أُصلحت | إغلاق الاتصالات، تصدير متدفق وآمن من المعادلات، رفض الشبكة دون إذن، إيقاف التصدير | `test_server_refuses_network_without_flag` + ٨ رحلات متصفح |
| 12 | أُصلحت | الذاكرة الفعلية للحاوية وقت التشغيل، إعادة القياس عند كل استلام، حجم الدفعة للعمال | (66) |
| 13 | أُصلحت (غير مثبتة على أوفيس) | `docpipe.ps1` مسار باورشيل فقط | `run-docpipe-ps.ps1` نجح في ٥٫١ و٧ على ويندوز CI (run 41169389853) |
| 14 | أُصلحت | قراءة حصص في الخلفية بمهلة، توحيد أسماء الاشتراك، أمر `herdr-goal quota`. **غير مثبت:** صيغ الأدوات الحقيقية | `TestSlowQuotaSourceDoesNotBlock`، `TestAccountAliasesShareOneBudget` |
| 15 | أُصلحت | اكتشاف الكتالوج تلقائيًا، بصمات كاملة مثبتة، احتواء المسارات، ميزانية سياق، سياسة الشركة دائمًا | `TestMandatoryPolicyAlwaysAndPinnedHashesEnforced`، `TestSkillPathsMustStayInsideTheCatalog`، `TestSkillBudgetBoundsTheBrief` |
| 16 | جزئيًا | البحث بوابة قبل الخطة وبعد فشلين؛ **غير منفذ:** تقييم المهارات مع/بدون على وكلاء حقيقيين | `TestResearchGates` |

## أدلة التشغيل
- Go: كل الاختبارات مع `-race`؛ التحور 54/54 (`app/tests/goal-mutations.py`).
- مسار المستندات: 42 اختبار بايثون؛ رحلات المتصفح 8/8؛ قياس 300 ألف صف = 54MB.
- CI لينكس وويندوز على كل دفع (`independent-goal-audit.yml`).
- التجربة الكاملة `run.sh` (٦ مهام، قتل، إلغاء، نتيجة متأخرة) خضراء؛ حملة الأعطال الكاملة بعد آخر إصلاح: انظر `tasks/goal-engine-report.md`.

## بوابات غير منفذة (ليست ناجحة)
- أوفيس حقيقي وملفات ناسكا، نسخة مستخدم مفتوحة، قتل العامل: `tools/device-harness/run-device-gates.ps1` على الجهاز المصرح.
- وكلاء حقيقيون مسجلون وحصصهم الفعلية وتشغيل ساعتين: يحتاج جهاز المالك وحساباته.
- تقييم المهارات مع/بدون على مهام فعلية.
