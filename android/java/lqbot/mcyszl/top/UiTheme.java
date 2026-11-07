package lqbot.mcyszl.top;

import android.app.Activity;
import android.content.Context;
import android.content.SharedPreferences;
import android.content.res.Configuration;

/**
 * 外观管理：主题可选「跟随系统 / 浅色 / 深色」，设置页修改后立即生效。
 * 页面颜色统一从这里取，避免系统深色模式切换后仍沿用旧颜色。
 */
public final class UiTheme {
    public static final String PREFS = "lqbot_prefs";
    public static final String KEY_THEME = "theme_mode";
    public static final String SYSTEM = "system";
    public static final String LIGHT = "light";
    public static final String DARK = "dark";

    private UiTheme() {
    }

    public static String mode(Context c) {
        return c.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
                .getString(KEY_THEME, SYSTEM);
    }

    public static boolean isDark(Context c) {
        String m = mode(c);
        if (DARK.equals(m)) {
            return true;
        }
        if (LIGHT.equals(m)) {
            return false;
        }
        int night = c.getResources().getConfiguration().uiMode
                & Configuration.UI_MODE_NIGHT_MASK;
        return night == Configuration.UI_MODE_NIGHT_YES;
    }

    /** 在 setContentView 之前调用，按选择的主题切换窗口底色与系统栏。 */
    public static void apply(Activity activity) {
        activity.setTheme(isDark(activity) ? R.style.AppTheme_Dark : R.style.AppTheme_Light);
    }

    public static int settingsBg(Context c) {
        return isDark(c) ? 0xFF0F172A : 0xFFF8FAFC;
    }

    public static int mainBg(Context c) {
        return isDark(c) ? 0xFF0F172A : 0xFFF3F4F6;
    }

    public static int title(Context c) {
        return isDark(c) ? 0xFFE5E7EB : 0xFF111827;
    }

    public static int hint(Context c) {
        return isDark(c) ? 0xFF94A3B8 : 0xFF6B7280;
    }
}
