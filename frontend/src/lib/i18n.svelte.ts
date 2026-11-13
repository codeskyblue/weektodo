export type Language = 'zh' | 'en';

const STORAGE_KEY = 'weektodo.language';

const en: Record<string, string> = {
	// Common
	'加载失败': 'Failed to load',
	'重试': 'Retry',
	'请求失败': 'Request failed',
	'取消': 'Cancel',
	'确定': 'OK',
	'关闭': 'Close',
	'保存设置': 'Save settings',
	'保存并关闭': 'Save and close',
	'保存中…': 'Saving…',
	'测试中…': 'Testing…',
	'测试连接': 'Test connection',
	'获取中…': 'Loading…',
	'获取列表': 'Get list',
	'没有匹配的模型': 'No matching models',
	'筛选 {count} 个模型…': 'Filter {count} models…',
	'已配置': 'Configured',
	'浅色': 'Light',
	'深色': 'Dark',
	'跟随系统': 'Follow system',
	'设置': 'Settings',
	'配置 AI 周报润色、WebDAV 同步、分类和界面外观。': 'Configure AI report polishing, WebDAV sync, categories and appearance.',
	'AI 周报润色': 'AI report polishing',
	'兼容 OpenAI 的 /chat/completions 接口。': 'OpenAI-compatible /chat/completions endpoint.',
	'模型': 'Model',
	'API Key': 'API Key',
	'已保存 {hint}，留空保持不变': 'Saved {hint}; leave blank to keep it',
	'Key 只保存在本机 settings.json（权限 0600），不会回传前端。':
		'The key is stored only in local settings.json (mode 0600) and is never sent back to the frontend.',
	'清除已保存的 Key': 'Clear saved key',
	'保存后将清除 Key': 'The key will be cleared on save',
	'外观': 'Appearance',
	'应用更新': 'App update',
	'当前版本 {version}': 'Current version {version}',
	'检查中…': 'Checking…',
	'检查更新': 'Check for updates',
	'发现 v{version}': 'v{version} available',
	'来源 {source}': 'Source: {source}',
	'安装 v{version}': 'Install v{version}',
	'正在更新…': 'Updating…',
	'查看 Release': 'View release',
	'当前是开发版本，无法检查更新。': 'This is a development build; updates are unavailable.',
	'当前已是最新版本。': 'You are up to date.',
	'正在下载…': 'Downloading…',
	'正在安装…': 'Installing…',
	'正在重启…': 'Restarting…',
	'准备中…': 'Preparing…',
	'更新已安装，正在重启…': 'Update installed; restarting…',
	'重启超时，请手动重新打开 WeekTodo。': 'Restart timed out. Please reopen WeekTodo manually.',
	'数据': 'Data',
	'复制路径': 'Copy path',
	'数据以 JSON 保存，每天首次保存时自动备份，保留最近 14 份每日备份。':
		'Data is saved as JSON. The first save of each day backs up the previous file; the latest 14 daily backups are kept.',
	'语言': 'Language',
	'切换语言': 'Switch language',
	'显示周数': 'Weeks shown',
	'显示 {count} 周': 'Show {count} weeks',
	'{count}周': '{count}w',
	'本周': 'This week',
	'添加任务…': 'Add a task…',
	'移到上一周': 'Move to previous week',
	'移到下一周': 'Move to next week',
	'已移到 {week}': 'Moved to {week}',
	'移动失败': 'Move failed',
	'拖动任务': 'Drag task',
	'界面语言': 'Interface language',
	'简体中文': 'Chinese (Simplified)',
	'English': 'English',

	// Header
	'WeekTodo 首页': 'WeekTodo home',
	'回到本周': 'Back to this week',
	'上一周': 'Previous week',
	'下一周': 'Next week',
	'加载中…': 'Loading…',
	'保存失败，重试': 'Save failed, retry',
	'周报': 'Report',
	'切换主题': 'Toggle theme',
	'项目主页': 'Project homepage',

	// Filters
	'全部': 'All',
	'今天': 'Today',
	'未完成': 'Open',
	'已完成': 'Done',
	'已废弃': 'Dropped',
	'任务筛选': 'Task filters',
	'今天还有 {count} 项待办': '{count} tasks left today',
	'{count} 项未安排日期': '{count} tasks without a date',
	'所有任务都已安排日期': 'All tasks have dates',
	'把上一周未完成的任务带入本周': 'Bring unfinished tasks from last week into this week',
	'带入上周': 'Bring from last week',

	// Tasks
	'设置日期': 'Set date',
	'清除日期': 'Clear date',
	'未安排': 'Unscheduled',
	'取消完成 {title}': 'Mark {title} incomplete',
	'完成 {title}': 'Complete {title}',
	'任务': 'Task',
	'从上一周带入': 'Carried from last week',
	'写点什么…': 'Write something…',
	'任务操作': 'Task actions',
	'添加于 {date}': 'Added {date}',
	'完成于 {date}': 'Completed {date}',
	'添加子任务': 'Add subtask',
	'上移': 'Move up',
	'下移': 'Move down',
	'恢复任务': 'Restore task',
	'废弃任务': 'Drop task',
	'删除任务': 'Delete task',

	// Empty state
	'没有符合筛选的任务': 'No tasks match this filter',
	'换个筛选条件看看。': 'Try a different filter.',
	'查看全部': 'View all',
	'这一周还没有任务': 'No tasks this week',
	'从一条小任务开始，或者把上周没做完的带过来。':
		'Start with a small task, or bring over unfinished tasks from last week.',
	'添加第一条': 'Add first task',
	'新建任务': 'New task',
	'带入上周未完成': 'Bring unfinished from last week',

	// Quick add
	'添加任务，按 Enter 保存…': 'Add a task, press Enter to save…',
	'不安排日期': 'No date',
	'添加': 'Add',
	'Enter 新建同级': 'Enter: new sibling',
	'行首 Tab 缩进': 'Tab at start: indent',
	'Shift+Tab 提升': 'Shift+Tab: outdent',
	'Alt+↑↓ 排序': 'Alt+↑↓: reorder',
	'Cmd/Ctrl+N 新建': 'Cmd/Ctrl+N: new task',
	'Cmd/Ctrl+Z 撤销': 'Cmd/Ctrl+Z: undo',
	'Cmd/Ctrl+Backspace 删除': 'Cmd/Ctrl+Backspace: delete',
	'粘贴多行导入': 'Paste multiple lines to import',

	// Report
	'周报已复制': 'Report copied',
	'周报-{id}.md': 'report-{id}.md',
	'AI 润色': 'AI polish',
	'完成 {count}': 'Done {count}',
	'废弃 {count}': 'Dropped {count}',
	'未完成 {count}': 'Open {count}',
	'复制 Markdown': 'Copy Markdown',
	'下载 Markdown': 'Download Markdown',
	'精简': 'Brief',
	'完整': 'Full',
	'恢复本地': 'Restore local',
	'生成中…': 'Generating…',
	'本周还没有可以写进周报的内容。': 'No completed work to include in the report yet.',
	'AI 生成的内容可能有偏差，发送前请检查。': 'AI output may be inaccurate; please review before sending.',

	// Application state / dialogs / toasts
	'（无标题任务）': '(Untitled task)',
	'保存失败': 'Save failed',
	'撤销': 'Undo',
	'完成父任务？': 'Complete parent task?',
	'「{title}」还有 {count} 个未完成的子任务。完成后会把这些子任务一起勾选。':
		'"{title}" has {count} unfinished subtasks. Completing it will check them all.',
	'全部完成': 'Complete all',
	'废弃任务？': 'Drop task?',
	'「{title}」及其 {count} 个子任务都会被废弃，且不计入完成率和周报。':
		'"{title}" and its {count} subtasks will be dropped and excluded from progress and reports.',
	'废弃': 'Drop',
	'删除任务？': 'Delete task?',
	'「{title}」及其 {count} 个子任务会被永久删除。':
		'"{title}" and its {count} subtasks will be permanently deleted.',
	'「{title}」会被永久删除。': '"{title}" will be permanently deleted.',
	'删除': 'Delete',
	'上周还没有记录': 'No record for last week',
	'没有可以带入的任务。': 'There are no tasks to bring over.',
	'上周未完成的任务已经全部带入本周': 'All unfinished tasks from last week have been brought over',
	'上周没有未完成的任务': 'Last week has no unfinished tasks',
	'… 还有 {count} 个': '… {count} more',
	'带入上周未完成任务': 'Bring unfinished tasks from last week',
	'将带入 {count} 个任务{skipped}：\n\n{preview}{more}':
		'Will bring over {count} tasks{skipped}:\n\n{preview}{more}',
	'，跳过 {count} 个已带入的任务': ', skipping {count} already carried tasks',
	'带入': 'Bring over',
	'已带入 {count} 个任务': 'Brought over {count} tasks',
	'跳过 {count} 个已带入的任务': 'Skipped {count} already carried tasks',
	'没有新的任务可带入': 'No new tasks to bring over',
	'带入失败': 'Failed to bring over tasks',
	'无法读取上周任务': "Could not read last week's tasks",
	'请先配置 AI 接口': 'Configure the AI endpoint first',
	'AI 周报已生成': 'AI report generated',
	'AI 润色失败': 'AI polish failed',
	'设置已保存': 'Settings saved',
	'保存设置失败': 'Failed to save settings',
	'AI 连接成功': 'AI connection successful',
	'AI 连接失败': 'AI connection failed',
	'请先填写 Base URL': 'Fill in the Base URL first',
	'请先填写 API Key': 'Fill in the API key first',
	'接口没有返回模型列表': 'The endpoint returned no models',
	'路径已复制': 'Path copied',
	'复制失败': 'Copy failed',
	'检查更新失败': 'Update check failed',

	// WebDAV
	'WebDAV 同步': 'WebDAV sync',
	'设置导航': 'Settings sections',
	'分类': 'Categories',
	'每个分类是独立的工作空间，任务、周报和带入互不混合。':
		'Each category is an independent workspace; tasks, reports and carry-over never mix.',
	'分类名称': 'Category name',
	'保存': 'Save',
	'归档': 'Archive',
	'恢复': 'Restore',
	'已归档': 'Archived',
	'新建分类': 'New category',
	'颜色': 'Color',
	'分类已保存': 'Category saved',
	'分类已创建': 'Category created',
	'分类已归档': 'Category archived',
	'分类已恢复': 'Category restored',
	'保存分类失败': 'Failed to save category',
	'将 weektodo.json 同步到远端目录，本地和远端会双向合并。':
		'Sync weektodo.json to a remote folder; local and remote changes are merged both ways.',
	'WebDAV 地址': 'WebDAV URL',
	'用户名': 'Username',
	'密码': 'Password',
	'远端目录': 'Remote folder',
	'相对于 WebDAV 地址，开头的 / 可省略；留空时同步到地址根目录。':
		'Relative to the WebDAV URL; a leading / is optional. Leave it empty to sync to the URL root.',
	'密码只保存在本机 settings.json（权限 0600），不会回传前端。':
		'The password is stored only in local settings.json (mode 0600) and is never sent back to the frontend.',
	'同步会按任务更新时间合并，较新的修改优先。':
		'Sync merges tasks by update time; newer edits win.',
	'立即同步': 'Sync now',
	'同步中…': 'Syncing…',
	'自动同步': 'Auto sync',
	'保存后自动同步，并每 5 分钟检查远端变化。':
		'Sync after saving and check for remote changes every 5 minutes.',
	'自动同步失败': 'Auto sync failed',
	'WebDAV 连接成功': 'WebDAV connection successful',
	'WebDAV 连接失败': 'WebDAV connection failed',
	'同步完成：{weeks} 周 / {tasks} 个任务': 'Sync complete · weeks: {weeks} / tasks: {tasks}',
	'同步失败': 'Sync failed',
	'清除已保存的密码': 'Clear saved password',
	'保存后将清除密码': 'The password will be cleared on save',

	// Dates
	'{year} 第 {week} 周': 'Week {week}, {year}'
};

function detectLanguage(): Language {
	if (typeof localStorage !== 'undefined') {
		const stored = localStorage.getItem(STORAGE_KEY);
		if (stored === 'zh' || stored === 'en') return stored;
	}
	if (typeof navigator !== 'undefined' && navigator.language.toLowerCase().startsWith('zh')) {
		return 'zh';
	}
	return 'en';
}

function interpolate(text: string, params?: Record<string, string | number>): string {
	if (!params) return text;
	return text.replace(/\{(\w+)\}/g, (match, name: string) =>
		params[name] === undefined ? match : String(params[name])
	);
}

class I18n {
	language = $state<Language>(detectLanguage());

	t(key: string, params?: Record<string, string | number>): string {
		const text = this.language === 'en' ? (en[key] ?? key) : key;
		return interpolate(text, params);
	}

	setLanguage(language: Language): void {
		this.language = language;
		if (typeof localStorage !== 'undefined') {
			localStorage.setItem(STORAGE_KEY, language);
		}
		if (typeof document !== 'undefined') {
			document.documentElement.lang = language === 'zh' ? 'zh-CN' : 'en';
		}
	}
}

export const i18n = new I18n();

export function t(key: string, params?: Record<string, string | number>): string {
	return i18n.t(key, params);
}
