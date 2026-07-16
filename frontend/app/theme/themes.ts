export const THEME_STORAGE_KEY = 'admin-theme';
export const ADMIN_THEME_EVENT = 'admin-theme-change';

export type AdminThemeId =
  | 'ocean'
  | 'sunset'
  | 'aurora'
  | 'neon'
  | 'sky'
  | 'forest'
  | 'coral'
  | 'amber'
  | 'graphite'
  | 'snow';

export type AdminTheme = {
  id: AdminThemeId;
  label: string;
  description: string;
  kind: 'gradient' | 'solid';
  mode: 'light' | 'dark';
  swatch: string;
  palette: {
    primary: string;
    primaryHover: string;
    primaryActive: string;
    onPrimary: string;
    accent: string;
    accentAlt: string;
    page: string;
    panel: string;
    elevated: string;
    hover: string;
    active: string;
    selected: string;
    text: string;
    textSecondary: string;
    textDisabled: string;
    border: string;
    focus: string;
    shadow: string;
    overlay: string;
    pageBackground: string;
    hero: string;
    charts: readonly [string, string, string, string, string];
  };
};

export const adminThemes: readonly AdminTheme[] = [
  {
    id: 'ocean',
    label: '深海流光',
    description: '蓝绿渐变',
    kind: 'gradient',
    mode: 'light',
    swatch: 'linear-gradient(135deg, #0f6cbd, #06b6d4 55%, #22c55e)',
    palette: {
      primary: '#0f6cbd', primaryHover: '#1684d8', primaryActive: '#0b5597', onPrimary: '#ffffff',
      accent: '#0891b2', accentAlt: '#16a34a', page: '#f2f8fc', panel: '#ffffff', elevated: '#f7fbfe',
      hover: '#e7f3fb', active: '#d5eaf7', selected: '#dff3f8', text: '#142634', textSecondary: '#526879',
      textDisabled: '#91a0aa', border: '#cadce7', focus: 'rgba(15, 108, 189, 0.26)',
      shadow: 'rgba(16, 54, 78, 0.12)', overlay: 'rgba(10, 32, 46, 0.48)',
      pageBackground: 'linear-gradient(135deg, #eef8ff 0%, #f8fbfd 50%, #eefcf7 100%)',
      hero: 'linear-gradient(120deg, #0f6cbd 0%, #0891b2 55%, #16a34a 100%)',
      charts: ['#0f6cbd', '#06b6d4', '#22c55e', '#f59e0b', '#e95678'],
    },
  },
  {
    id: 'sunset',
    label: '日落熔金',
    description: '珊瑚金渐变',
    kind: 'gradient',
    mode: 'light',
    swatch: 'linear-gradient(135deg, #d9485f, #f97316 55%, #facc15)',
    palette: {
      primary: '#d9485f', primaryHover: '#e85c70', primaryActive: '#b93249', onPrimary: '#ffffff',
      accent: '#ea580c', accentAlt: '#ca8a04', page: '#fff7f3', panel: '#ffffff', elevated: '#fffaf7',
      hover: '#fff0ea', active: '#ffe1d7', selected: '#ffe8df', text: '#3b2024', textSecondary: '#76565b',
      textDisabled: '#ae9296', border: '#ebcdc7', focus: 'rgba(217, 72, 95, 0.25)',
      shadow: 'rgba(99, 42, 30, 0.12)', overlay: 'rgba(54, 24, 20, 0.48)',
      pageBackground: 'linear-gradient(135deg, #fff1ee 0%, #fffaf5 52%, #fff9df 100%)',
      hero: 'linear-gradient(120deg, #c93455 0%, #f26b38 55%, #d49b07 100%)',
      charts: ['#d9485f', '#f97316', '#eab308', '#0f9f8f', '#6d5bd0'],
    },
  },
  {
    id: 'aurora',
    label: '极光森林',
    description: '青绿靛渐变',
    kind: 'gradient',
    mode: 'light',
    swatch: 'linear-gradient(135deg, #047857, #0891b2 55%, #4f46e5)',
    palette: {
      primary: '#047857', primaryHover: '#059669', primaryActive: '#065f46', onPrimary: '#ffffff',
      accent: '#0891b2', accentAlt: '#4f46e5', page: '#f2faf7', panel: '#ffffff', elevated: '#f7fcfa',
      hover: '#e5f5ef', active: '#d1ece2', selected: '#dcf4ee', text: '#16332d', textSecondary: '#526e68',
      textDisabled: '#91a59f', border: '#c8ded7', focus: 'rgba(4, 120, 87, 0.26)',
      shadow: 'rgba(17, 74, 59, 0.12)', overlay: 'rgba(8, 41, 33, 0.5)',
      pageBackground: 'linear-gradient(135deg, #edfbf5 0%, #f7fcfb 48%, #eef2ff 100%)',
      hero: 'linear-gradient(120deg, #047857 0%, #078b9c 55%, #4f46e5 100%)',
      charts: ['#047857', '#06b6d4', '#4f46e5', '#d97706', '#e44d78'],
    },
  },
  {
    id: 'neon',
    label: '霓虹夜航',
    description: '深色霓虹渐变',
    kind: 'gradient',
    mode: 'dark',
    swatch: 'linear-gradient(135deg, #0f172a, #0891b2 52%, #a3e635)',
    palette: {
      primary: '#22d3ee', primaryHover: '#67e8f9', primaryActive: '#06b6d4', onPrimary: '#082127',
      accent: '#a3e635', accentAlt: '#f472b6', page: '#080d15', panel: '#111923', elevated: '#172330',
      hover: '#1b2b38', active: '#223746', selected: '#123846', text: '#eefcff', textSecondary: '#a9c2c9',
      textDisabled: '#687f86', border: '#2b414d', focus: 'rgba(34, 211, 238, 0.35)',
      shadow: 'rgba(0, 0, 0, 0.48)', overlay: 'rgba(0, 0, 0, 0.76)',
      pageBackground: 'linear-gradient(135deg, #071018 0%, #0d1721 55%, #111a18 100%)',
      hero: 'linear-gradient(120deg, #087f93 0%, #0e7490 46%, #4d7c0f 100%)',
      charts: ['#22d3ee', '#a3e635', '#f472b6', '#fbbf24', '#818cf8'],
    },
  },
  {
    id: 'sky',
    label: '商务蓝',
    description: '经典蓝纯色',
    kind: 'solid',
    mode: 'light',
    swatch: '#2563eb',
    palette: {
      primary: '#2563eb', primaryHover: '#3b76ee', primaryActive: '#1d4ed8', onPrimary: '#ffffff',
      accent: '#0891b2', accentAlt: '#16a34a', page: '#f4f7fb', panel: '#ffffff', elevated: '#f8fafc',
      hover: '#edf3ff', active: '#dce8ff', selected: '#e4edff', text: '#172033', textSecondary: '#536176',
      textDisabled: '#98a2b3', border: '#d0d9e6', focus: 'rgba(37, 99, 235, 0.25)',
      shadow: 'rgba(15, 23, 42, 0.12)', overlay: 'rgba(15, 23, 42, 0.48)',
      pageBackground: '#f4f7fb', hero: '#2563eb',
      charts: ['#2563eb', '#0891b2', '#16a34a', '#d97706', '#dc4c64'],
    },
  },
  {
    id: 'forest',
    label: '松林绿',
    description: '沉静绿纯色',
    kind: 'solid',
    mode: 'light',
    swatch: '#15803d',
    palette: {
      primary: '#15803d', primaryHover: '#18964a', primaryActive: '#116530', onPrimary: '#ffffff',
      accent: '#0f766e', accentAlt: '#b7791f', page: '#f4f8f5', panel: '#ffffff', elevated: '#f8fbf9',
      hover: '#eaf5ed', active: '#d9ebde', selected: '#e1f1e6', text: '#193326', textSecondary: '#566b60',
      textDisabled: '#96a69d', border: '#cddbd2', focus: 'rgba(21, 128, 61, 0.25)',
      shadow: 'rgba(19, 68, 37, 0.11)', overlay: 'rgba(12, 42, 24, 0.5)',
      pageBackground: '#f4f8f5', hero: '#15803d',
      charts: ['#15803d', '#0f766e', '#2563eb', '#d97706', '#cf3f5b'],
    },
  },
  {
    id: 'coral',
    label: '珊瑚红',
    description: '活力红纯色',
    kind: 'solid',
    mode: 'light',
    swatch: '#d83a4e',
    palette: {
      primary: '#d83a4e', primaryHover: '#e65364', primaryActive: '#b72b3d', onPrimary: '#ffffff',
      accent: '#ea580c', accentAlt: '#0f8b8d', page: '#fff6f6', panel: '#ffffff', elevated: '#fffafa',
      hover: '#ffedef', active: '#ffdade', selected: '#ffe4e7', text: '#3a2023', textSecondary: '#76575b',
      textDisabled: '#ad9295', border: '#e8cdd0', focus: 'rgba(216, 58, 78, 0.25)',
      shadow: 'rgba(94, 30, 39, 0.11)', overlay: 'rgba(53, 17, 22, 0.48)',
      pageBackground: '#fff6f6', hero: '#d83a4e',
      charts: ['#d83a4e', '#ea580c', '#0f8b8d', '#2563eb', '#ca8a04'],
    },
  },
  {
    id: 'amber',
    label: '琥珀金',
    description: '暖金纯色',
    kind: 'solid',
    mode: 'light',
    swatch: '#b45309',
    palette: {
      primary: '#b45309', primaryHover: '#cb650f', primaryActive: '#92400e', onPrimary: '#ffffff',
      accent: '#ca8a04', accentAlt: '#0f766e', page: '#fff9ef', panel: '#ffffff', elevated: '#fffcf6',
      hover: '#fff3d8', active: '#ffe7b5', selected: '#ffedc8', text: '#382a19', textSecondary: '#75644d',
      textDisabled: '#a99b88', border: '#e5d6bd', focus: 'rgba(180, 83, 9, 0.25)',
      shadow: 'rgba(91, 57, 18, 0.11)', overlay: 'rgba(49, 32, 12, 0.48)',
      pageBackground: '#fff9ef', hero: '#b45309',
      charts: ['#b45309', '#ca8a04', '#0f766e', '#2563eb', '#c24178'],
    },
  },
  {
    id: 'graphite',
    label: '石墨暗夜',
    description: '深灰纯色',
    kind: 'solid',
    mode: 'dark',
    swatch: '#20242d',
    palette: {
      primary: '#69a7ff', primaryHover: '#91c0ff', primaryActive: '#478de8', onPrimary: '#08111f',
      accent: '#34d399', accentAlt: '#f59e0b', page: '#0f1115', panel: '#171a21', elevated: '#20242d',
      hover: '#252b36', active: '#2d3748', selected: '#17345f', text: '#f5f7fa', textSecondary: '#b8c0cc',
      textDisabled: '#727b88', border: '#3b4350', focus: 'rgba(105, 167, 255, 0.38)',
      shadow: 'rgba(0, 0, 0, 0.46)', overlay: 'rgba(0, 0, 0, 0.72)',
      pageBackground: '#0f1115', hero: '#295f9e',
      charts: ['#69a7ff', '#34d399', '#f59e0b', '#f472b6', '#a78bfa'],
    },
  },
  {
    id: 'snow',
    label: '黑白简约',
    description: '中性灰纯色',
    kind: 'solid',
    mode: 'light',
    swatch: '#3f444c',
    palette: {
      primary: '#3f444c', primaryHover: '#555b64', primaryActive: '#292d33', onPrimary: '#ffffff',
      accent: '#0f766e', accentAlt: '#c2415d', page: '#f5f6f7', panel: '#ffffff', elevated: '#fafafa',
      hover: '#eff0f2', active: '#e2e4e7', selected: '#e7e9ec', text: '#202329', textSecondary: '#626872',
      textDisabled: '#9ba0a8', border: '#d5d8dc', focus: 'rgba(63, 68, 76, 0.24)',
      shadow: 'rgba(24, 28, 34, 0.11)', overlay: 'rgba(24, 28, 34, 0.48)',
      pageBackground: '#f5f6f7', hero: '#3f444c',
      charts: ['#3f444c', '#0f766e', '#c2415d', '#2563eb', '#b7791f'],
    },
  },
];

export const DEFAULT_THEME_ID: AdminThemeId = 'ocean';

const legacyThemeAliases: Record<string, AdminThemeId> = {
  light: 'sky',
  dark: 'graphite',
  pink: 'sunset',
};

export function resolveThemeId(value: string | null | undefined): AdminThemeId {
  if (!value) return DEFAULT_THEME_ID;
  const alias = legacyThemeAliases[value];
  if (alias) return alias;
  return adminThemes.some((theme) => theme.id === value) ? (value as AdminThemeId) : DEFAULT_THEME_ID;
}

export function getAdminTheme(themeId: string | null | undefined): AdminTheme {
  const resolved = resolveThemeId(themeId);
  return adminThemes.find((theme) => theme.id === resolved) ?? adminThemes[0];
}

export function getThemeCssVariables(theme: AdminTheme): Record<string, string> {
  const { palette } = theme;
  return {
    '--background': palette.page,
    '--foreground': palette.text,
    '--card': palette.panel,
    '--card-foreground': palette.text,
    '--popover': palette.panel,
    '--popover-foreground': palette.text,
    '--primary': palette.primary,
    '--primary-foreground': palette.onPrimary,
    '--secondary': palette.elevated,
    '--secondary-foreground': palette.text,
    '--muted': palette.elevated,
    '--muted-foreground': palette.textSecondary,
    '--accent': palette.hover,
    '--accent-foreground': palette.text,
    '--border': palette.border,
    '--input': palette.border,
    '--ring': palette.primary,
    '--sidebar': palette.panel,
    '--sidebar-foreground': palette.text,
    '--sidebar-primary': palette.primary,
    '--sidebar-primary-foreground': palette.onPrimary,
    '--sidebar-accent': palette.hover,
    '--sidebar-accent-foreground': palette.text,
    '--sidebar-border': palette.border,
    '--sidebar-ring': palette.primary,
    '--surface': palette.panel,
    '--control-hover': palette.hover,
    '--control-active': palette.active,
    '--primary-hover': palette.primaryHover,
    '--primary-active': palette.primaryActive,
    '--focus-border': palette.primary,
    '--selected-bg': palette.selected,
    '--selected-fg': palette.primary,
    '--text-tertiary': palette.textSecondary,
    '--accent-blue': palette.primary,
    '--accent-pink': palette.accent,
    '--surface-page': palette.page,
    '--surface-panel': palette.panel,
    '--surface-elevated': palette.elevated,
    '--surface-hover': palette.hover,
    '--surface-active': palette.active,
    '--surface-selected': palette.selected,
    '--text-primary': palette.text,
    '--text-secondary': palette.textSecondary,
    '--text-disabled': palette.textDisabled,
    '--border-subtle': palette.border,
    '--focus-ring': palette.focus,
    '--shadow-color': palette.shadow,
    '--overlay-color': palette.overlay,
    '--theme-primary': palette.primary,
    '--theme-on-primary': palette.onPrimary,
    '--theme-accent': palette.accent,
    '--theme-accent-alt': palette.accentAlt,
    '--theme-page-background': palette.pageBackground,
    '--theme-hero': palette.hero,
    '--chart-1': palette.charts[0],
    '--chart-2': palette.charts[1],
    '--chart-3': palette.charts[2],
    '--chart-4': palette.charts[3],
    '--chart-5': palette.charts[4],
  };
}

export function applyAdminTheme(themeId: AdminThemeId, persist = true) {
  if (typeof document === 'undefined') return;
  const theme = getAdminTheme(themeId);
  const root = document.documentElement;
  root.dataset.theme = theme.id;
  root.dataset.themeMode = theme.mode;
  root.classList.toggle('dark', theme.mode === 'dark');
  root.style.colorScheme = theme.mode;
  Object.entries(getThemeCssVariables(theme)).forEach(([name, value]) => root.style.setProperty(name, value));
  if (persist && typeof window !== 'undefined') window.localStorage.setItem(THEME_STORAGE_KEY, theme.id);
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent(ADMIN_THEME_EVENT, { detail: { themeId: theme.id } }));
  }
}

const bootstrapThemes = Object.fromEntries(
  adminThemes.map((theme) => [theme.id, { mode: theme.mode, variables: getThemeCssVariables(theme) }]),
);

export const ADMIN_THEME_BOOTSTRAP_SCRIPT = `(() => {
  try {
    const themes = ${JSON.stringify(bootstrapThemes)};
    const aliases = ${JSON.stringify(legacyThemeAliases)};
    const stored = window.localStorage.getItem(${JSON.stringify(THEME_STORAGE_KEY)});
    const id = themes[stored] ? stored : (aliases[stored] || ${JSON.stringify(DEFAULT_THEME_ID)});
    const theme = themes[id];
    const root = document.documentElement;
    root.dataset.theme = id;
    root.dataset.themeMode = theme.mode;
    root.classList.toggle('dark', theme.mode === 'dark');
    root.style.colorScheme = theme.mode;
    Object.entries(theme.variables).forEach(([name, value]) => root.style.setProperty(name, value));
  } catch {}
})();`;
