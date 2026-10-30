# رد على المراجعة المستقلة لمنظومة الأهداف (PR #16) — ٣٠ أكتوبر ٢٠٢٦

الفرع: `ccr-efd59209-8jk5t6`. النسخة المفحوصة في المراجعة: `cd846676`؛ الاختبارات الستة أُعيد تشغيلها
على `3b6d8b1` قبل أي إصلاح وفشلت الستة كلها (نفس أسباب لينكس وويندوز في المراجعة).
كل إصلاح هنا له اختبار سلوك يفشل على الكود السابق (أُثبت بتشغيله على نسخة سابقة) وينجح بعده، وتجربة
تحور تكسر القاعدة فيفشل اختبارها: `python3 app/tests/goal-mutations.py` → 45/45.
سير CI (`independent-goal-audit.yml`) يعمل الآن على هذا الفرع على لينكس وويندوز مع `-race`.

الحالة: **أُصلحت** (مع الدليل) / **جزئيًا** (ما بقي مذكور) / **مفتوحة**.

| # | الحالة | ما تغيّر | الدليل |
|---|---|---|---|
| ١ | أُصلحت | قفل نظام تشغيل `runner.lck` (flock / LockFileEx) ممسوك طوال عمر المشرف، يزول بموته؛ `runnerAlive` يختبر القفل لا رقم العملية؛ جيل مشرف `runner_gen` يمنع الكاتب القديم من أي كتابة (يتوقف قبل أي أثر لأن النية تُحفظ قبل التنفيذ). | `TestRunnerLockIsExclusiveAndReleased`، `TestOldRunnerGenerationCannotWrite`؛ e2e: المشرف الثاني رُفض "runner.lck is locked". |
| ٢ | أُصلحت | `perform` يرفض الإرسال لوكيل غير مملوك للهدف أو منتهٍ أو خرج (`!Exists`/`exited`)، إضافة لفحص طلب السر. | `TestIndependentAuditExitedAgentGetsNoInput` |
| ٣ | جزئيًا | المستهلك يستمد الدور من رمز (token) لا من `by`: مفتاح المالك `keys/owner` بصلاحية 0600، ورمز لكل قائد يُلغى عند استبداله، والمنفذون بلا رمز؛ جدول صلاحيات عند الاستهلاك. **باقٍ:** عملية بنفس حساب المستخدم تستطيع قراءة مفتاح المالك؛ العزل الحقيقي يتطلب تشغيل الوكلاء بحساب نظام آخر. | `TestInboxRoleComesFromTokenNotClaim`، `TestLeadTokenProvesLeadOnly`، `TestIndependentAuditConsumerRejectsDelegateStop` |
| ٤ | أُصلحت | مالك واحد لنسخة التكامل: الدمج والتحقق والتراجع وظائف "main" لا تعمل اثنتان معًا؛ التراجع صار طابورًا يُنفذ عند فراغ النسخة؛ الاستعادة تُلغي دمجًا معلقًا. | `TestIntegrationWorktreeHasOneOwner`، `TestMergeConflictGoesBackToWriterAndRevertKeepsOthers` |
| ٥ | أُصلحت | كل وظيفة خلفية بسياق إلغاء؛ الإلغاء والمهلة يقتلان شجرة العمليات (مجموعة عمليات / `taskkill /T`)؛ مفتاح الوظيفة يحمل جيل المهمة؛ نتيجة الفحص تُقبل فقط إن كان فرع المهمة على نفس الإيداع المفحوص؛ الإيقاف/التعديل يلغي الوظائف. | `TestCancelAndTimeoutKillTheProcessTree`، `TestStaleCheckResultIsDropped`، `TestCheckOfAnOlderCommitIsDropped` |
| ٦ | أُصلحت | المراجعة تُقبل فقط بـ `status=done` و`verdict=pass`؛ الحكم مربوط بالإيداع المراجَع ويُدمج ذلك الإيداع بالضبط؛ تغيّر الفرع بعد بدء المراجعة يعيدها؛ بديل fallback من عائلة الكاتب لا يراجع. | `TestIndependentAuditBlockedReviewCannotApprove`، `TestReviewBoundToTheReviewedCommit` |
| ٧ | أُصلحت | الثقة تُقرر على المسار الظاهر في الطلب نفسه بعد حل الروابط؛ طلب بلا مسار يُقبل فقط كطلب بدء لوكيل لم يبدأ بعد في مجلده. | `TestTrustUsesTheFolderOnScreen`، `TestTrustOutsideProjectGoesToOwner` |
| ٨ | أُصلحت | لا اختيار افتراضي أعمى: قائمة فيها خيار خطر (دفع، حذف، نشر، فرض…) تُصعّد ولا تُجاب؛ المستشار يعمل في الخلفية والمراقبة مستمرة، ويُرسل اختياره فقط إن بقيت الشاشة نفسها. | `TestRiskyMenuIsNotAnsweredByDefault`، `TestDeciderRunsInTheBackground` |
| ٩ | أُصلحت | الإيقاف يُؤكَّد بالملاحظة قبل حفظ العمل؛ إيقاف غير مؤكد يُبقي الملكية والعدّ (`stop-pending`) ولا يُعاد التشغيل فوقه؛ حد حساب يوقف مهمته فقط والوكلاء السليمون يكملون. | `TestUnconfirmedStopKeepsOwnership`، `TestAccountLimitDoesNotStopHealthyAgents` |
| ١٠ | أُصلحت | أُزيل `clean -fdx`؛ تُحذف فقط الملفات التي يسميها git كعائق للدمج داخل نسخة المحرك؛ كل أمر git بمهلة؛ المخرجات محدودة بذيل ثابت الحجم؛ القتل لشجرة العمليات. | `TestMergeKeepsUntrackedDependencies`، `TestCheckOutputIsBounded` |
| ١١ | أُصلحت | الملغاة/الفاشلة لا تحقق الاعتماديات؛ التابع ينتظر إعادة تخطيط (حدث `dependency-broken`)؛ الدورات مرفوضة. | `TestIndependentAuditCancelledPrerequisiteIsNotSuccess`، `TestCancelledPrerequisiteWaitsForReplanThenRuns`، `TestDependencyCycleRejectedWithoutEffect` |
| ١٢ | أُصلحت | تطبيع المسارات (شرطة، `./`، حالة الأحرف على ويندوز) وملكية المجلد لما بداخله. **باقٍ:** الحجز ليس عزلًا، العزل هو نسخة العمل المنفصلة. | `TestIndependentAuditDirectoryOwnershipIncludesChildren`، `TestPathReservationsNormalised` |
| ١٣ | أُصلحت | تحقق كامل قبل أي أثر، ثم إيقاف وانتقال. | `TestIndependentAuditInvalidUpdateLeavesWorkerRunning` |
| ١٤ | أُصلحت | التقدم من أحداث المنتج فقط (نتائج، فحوص، مراجعات، دمج، إعادة تخطيط)، لا من إعادة التشغيل والتنبيه؛ ميزانية فشل عبر إعادات التخطيط لا يصفّرها القائد؛ القائد البديل يُبلّغ بالمشكلات المفتوحة. اكتُشف في الجولة ٥ من التشغيل الممتد. | `TestRestartLoopIsNotProgress`، `TestFailureBudgetSurvivesReplans`، `TestNewLeadIsToldAboutOpenProblems` |
| ١٥ | جزئيًا | الأحداث تُزال مكرراتها بالرقم ويُتخطى السطر التالف في أي موضع؛ الحالة ملف واحد ذري. **باقٍ:** لا قاعدة بيانات للسجل ولا اختبار امتلاء قرص أو انقطاع كهرباء على ويندوز؛ إعادة الإرسال بعد عطل تُترك لقرار المراقب من الشاشة (لا وعد بالتنفيذ مرة واحدة). | `TestEventsDeduplicateAndSkipDamagedLines`، `TestLedgerSurvivesTornWrites` |
| ١٦ | جزئيًا | `panes.json` التالف خطأ صريح يوقف التشغيل؛ رفض الأسطر الجديدة ومحارف التحكم في المعاملات. **باقٍ:** لا اختبار عقد مع هيردر الحقيقي (لم يُبنَ هنا: يحتاج Zig 0.16 المحجوب). | `TestHerdrBackendRefusesDamagedPaneMapAndNewlines`، `TestHerdrBackendCommandsAndPaneMemory` |
| ١٧ | جزئيًا | ميزانية عمليات `max_procs` تحسب القائد والوكلاء والوظائف، والقائد يبدأ أولًا؛ بوابة الذاكرة تحجز تقدير وكيل؛ فحص دخول اختياري (`probe`) يستبعد أداة غير مسجلة الدخول. **باقٍ:** اختبارات قبول مستقلة محفوظة قبل التنفيذ مسؤولية القائد (المهارة)، والمراجع ما زال يرى نسخة عمل الكاتب (للقراءة بتعليمات فقط). | `TestProcessBudgetCountsLeadAndJobs`، `TestProbeDropsAProfileThatDoesNotAnswer` |
| ١٨ | أُصلحت | بصمة نسخة العمل (الملفات تتغير) تعد تقدمًا؛ وقت التجديد بالمنطقة الزمنية المطبوعة. | `TestWorktreeChangesAreProgress`، `TestResetTimeUsesThePrintedZone` |

## ما وجده التشغيل الممتد ولم يكن في المراجعة
- قائد سليم يُقتل من الخارج كان يستهلك سقف إعادة التشغيل فيقف الهدف عند صاحب الجهاز، ثم يترك وكلاء بلا مراقبة (الجولة ٨) — أُصلح (`TestHealthyAgentKilledManyTimesKeepsRestarting`، `TestNeedsOwnerLeavesNoUnwatchedAgents`).
- الحفظ التلقائي كان يضيف الذاكرة المؤقتة والأسرار (`git add -A`) — صار انتقائيًا (`TestAutosaveSkipsCachesSecretsAndHugeFiles`).
- سباق بيانات في اختبار كشفه `-race` على CI — أُصلح.

## ما لم يُثبت
- أي تشغيل بوكلاء حقيقيين (كلود/كودكس) أو داخل ألواح هيردر حقيقية: غير متاح في هذه الحاوية. كل تجارب التكامل بوكيل مبرمج (`app/tests/goalagent`) نوعين مختلفين بالاسم والإعداد، ولا تثبت تنوع النماذج ولا استقلال مراجعة حقيقي.
- ناسكا وأوفيس COM: لا يوجد أوفيس ولا ويندوز هنا؛ عامل COM يُفحص تحليله على ويندوز في CI فقط.
- تشغيل حقيقي لساعتين: التشغيل الممتد هنا ٣٠–٣٥ دقيقة بمحاكاة وفوضى، لا ساعتين بوكلاء حقيقيين.
