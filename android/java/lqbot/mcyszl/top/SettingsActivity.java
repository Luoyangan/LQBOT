package lqbot.mcyszl.top;

import android.app.Activity;
import android.app.AlertDialog;
import android.content.ComponentName;
import android.content.DialogInterface;
import android.content.Intent;
import android.content.SharedPreferences;
import android.graphics.Color;
import android.net.Uri;
import android.os.Build;
import android.os.Bundle;
import android.os.Environment;
import android.os.PowerManager;
import android.provider.Settings;
import android.util.TypedValue;
import android.view.View;
import android.view.ViewGroup;
import android.view.WindowManager;
import android.widget.Button;
import android.widget.CompoundButton;
import android.widget.LinearLayout;
import android.widget.ScrollView;
import android.widget.Switch;
import android.widget.TextView;
import android.widget.Toast;

/** 独立设置页：屏幕常亮、悬浮窗、以及保活相关开关。 */
public class SettingsActivity extends Activity {
    private static final String PREFS = "lqbot_prefs";
    private static final String KEY_SCREEN_ON = "screen_on";
    private static final String KEY_OVERLAY = "overlay";
    private static final String KEY_OVERLAY_OPAQUE = "overlay_opaque";
    private static final String KEY_BOOT = "boot_autostart";
    private static final String KEY_WATCHDOG = "watchdog";
    private static final String KEY_GUARD = "guard";
    private static final String KEY_ONLY_CHARGING = "only_charging";
    private static final String KEY_ONLY_WIFI = "only_wifi";
    private static final String KEY_DAILY_RESTART = "daily_restart";
    private static final String KEY_RESTART_HOUR = "restart_hour";
    private static final String KEY_RESTART_MIN = "restart_min";
    private static final String KEY_OVERLAY_CONTENT = "overlay_content";
    private static final String KEY_LIVE_UPDATE = "live_update";

    private SharedPreferences prefs;
    private Switch screenSwitch;
    private Switch overlaySwitch;
    private Switch opaqueSwitch;
    private Button contentButton;
    private boolean darkUi;

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        prefs = getSharedPreferences(PREFS, MODE_PRIVATE);
        UiTheme.apply(this);
        darkUi = UiTheme.isDark(this);
        buildUi();
    }

    @Override
    protected void onResume() {
        super.onResume();
        if (darkUi != UiTheme.isDark(this)) {
            recreate();
            return;
        }
        // 从悬浮窗授权页返回：已授权则立即应用
        if (overlaySwitch.isChecked() && Settings.canDrawOverlays(this)) {
            sendOverlayConfig();
        }
    }

    private void buildUi() {
        ScrollView scroll = new ScrollView(this);
        scroll.setBackgroundColor(UiTheme.settingsBg(this));
        LinearLayout root = new LinearLayout(this);
        root.setOrientation(LinearLayout.VERTICAL);
        root.setPadding(dp(20), dp(20), dp(20), dp(20));
        scroll.addView(root, new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.WRAP_CONTENT));

        TextView title = new TextView(this);
        title.setText(R.string.settings_title);
        title.setTextSize(TypedValue.COMPLEX_UNIT_SP, 20);
        title.setTextColor(UiTheme.title(this));
        title.setTypeface(title.getTypeface(), android.graphics.Typeface.BOLD);
        root.addView(title, new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.WRAP_CONTENT));

        TextView hint = new TextView(this);
        hint.setText(R.string.settings_hint);
        hint.setTextSize(TypedValue.COMPLEX_UNIT_SP, 12);
        hint.setTextColor(UiTheme.hint(this));
        hint.setPadding(0, dp(6), 0, dp(12));
        root.addView(hint);

        // 外观：主题（跟随系统 / 浅色 / 深色）
        final Button themeButton = new Button(this);
        themeButton.setAllCaps(false);
        themeButton.setText(getString(R.string.theme_current, themeLabel(UiTheme.mode(this))));
        root.addView(themeButton, new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, dp(44)));
        themeButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                final String[] values = {UiTheme.SYSTEM, UiTheme.LIGHT, UiTheme.DARK};
                String[] labels = {
                        getString(R.string.theme_system),
                        getString(R.string.theme_light),
                        getString(R.string.theme_dark)};
                new AlertDialog.Builder(SettingsActivity.this)
                        .setTitle(R.string.option_theme)
                        .setItems(labels, new DialogInterface.OnClickListener() {
                            @Override
                            public void onClick(DialogInterface dialog, int which) {
                                prefs.edit().putString(UiTheme.KEY_THEME, values[which]).apply();
                                recreate();
                            }
                        })
                        .show();
            }
        });

        screenSwitch = addSwitchRow(root, R.string.option_screen_on);
        overlaySwitch = addSwitchRow(root, R.string.option_overlay);
        opaqueSwitch = addSwitchRow(root, R.string.option_overlay_opaque);
        Switch bootSwitch = addSwitchRow(root, R.string.option_boot_autostart);
        Switch watchdogSwitch = addSwitchRow(root, R.string.option_watchdog);
        Switch guardSwitch = addSwitchRow(root, R.string.option_guard);

        screenSwitch.setChecked(prefs.getBoolean(KEY_SCREEN_ON, false));
        overlaySwitch.setChecked(prefs.getBoolean(KEY_OVERLAY, false));
        opaqueSwitch.setChecked(prefs.getBoolean(KEY_OVERLAY_OPAQUE, true));
        bootSwitch.setChecked(prefs.getBoolean(KEY_BOOT, true));
        watchdogSwitch.setChecked(prefs.getBoolean(KEY_WATCHDOG, true));
        guardSwitch.setChecked(prefs.getBoolean(KEY_GUARD, true));

        applyScreenOn(screenSwitch.isChecked());
        opaqueSwitch.setEnabled(overlaySwitch.isChecked());

        screenSwitch.setOnCheckedChangeListener(new CompoundButton.OnCheckedChangeListener() {
            @Override
            public void onCheckedChanged(CompoundButton buttonView, boolean isChecked) {
                prefs.edit().putBoolean(KEY_SCREEN_ON, isChecked).apply();
                applyScreenOn(isChecked);
            }
        });
        overlaySwitch.setOnCheckedChangeListener(new CompoundButton.OnCheckedChangeListener() {
            @Override
            public void onCheckedChanged(CompoundButton buttonView, boolean isChecked) {
                prefs.edit().putBoolean(KEY_OVERLAY, isChecked).apply();
                opaqueSwitch.setEnabled(isChecked);
                if (contentButton != null) {
                    contentButton.setEnabled(isChecked);
                }
                if (isChecked && !Settings.canDrawOverlays(SettingsActivity.this)) {
                    requestOverlayPermission();
                    return;
                }
                sendOverlayConfig();
            }
        });
        opaqueSwitch.setOnCheckedChangeListener(new CompoundButton.OnCheckedChangeListener() {
            @Override
            public void onCheckedChanged(CompoundButton buttonView, boolean isChecked) {
                prefs.edit().putBoolean(KEY_OVERLAY_OPAQUE, isChecked).apply();
                if (overlaySwitch.isChecked()) {
                    sendOverlayConfig();
                }
            }
        });

        // 悬浮窗显示内容（多选）
        contentButton = new Button(this);
        contentButton.setAllCaps(false);
        contentButton.setText(getString(R.string.overlay_content_current,
                contentLabel(prefs.getString(KEY_OVERLAY_CONTENT, "status,uptime"))));
        contentButton.setEnabled(overlaySwitch.isChecked());
        contentButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                final String[] keys = {"status", "uptime", "port", "accounts", "stats"};
                String[] labels = {
                        getString(R.string.overlay_content_status),
                        getString(R.string.overlay_content_uptime),
                        getString(R.string.overlay_content_port),
                        getString(R.string.overlay_content_accounts),
                        getString(R.string.overlay_content_stats)};
                final boolean[] checked = {
                        currentContent().contains("status"),
                        currentContent().contains("uptime"),
                        currentContent().contains("port"),
                        currentContent().contains("accounts"),
                        currentContent().contains("stats")};
                new AlertDialog.Builder(SettingsActivity.this)
                        .setTitle(R.string.option_overlay_content)
                        .setMultiChoiceItems(labels, checked,
                                new DialogInterface.OnMultiChoiceClickListener() {
                                    @Override
                                    public void onClick(DialogInterface dialog, int which, boolean isChecked) {
                                        checked[which] = isChecked;
                                    }
                                })
                        .setPositiveButton(android.R.string.ok, new DialogInterface.OnClickListener() {
                            @Override
                            public void onClick(DialogInterface dialog, int which) {
                                StringBuilder sb = new StringBuilder();
                                for (int i = 0; i < keys.length; i++) {
                                    if (checked[i]) {
                                        if (sb.length() > 0) {
                                            sb.append(',');
                                        }
                                        sb.append(keys[i]);
                                    }
                                }
                                String value = sb.length() > 0 ? sb.toString() : "status";
                                prefs.edit().putString(KEY_OVERLAY_CONTENT, value).apply();
                                contentButton.setText(getString(R.string.overlay_content_current,
                                        contentLabel(value)));
                                sendOverlayConfig();
                            }
                        })
                        .setNegativeButton(android.R.string.cancel, null)
                        .show();
            }
        });
        root.addView(contentButton, new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, dp(44)));

        // Android 16+ 系统实时活动（灵动岛/原子岛）
        if (Build.VERSION.SDK_INT >= 36) {
            Switch liveSwitch = addSwitchRow(root, R.string.option_live_update);
            liveSwitch.setChecked(prefs.getBoolean(KEY_LIVE_UPDATE, true));
            liveSwitch.setOnCheckedChangeListener(new CompoundButton.OnCheckedChangeListener() {
                @Override
                public void onCheckedChanged(CompoundButton buttonView, boolean isChecked) {
                    prefs.edit().putBoolean(KEY_LIVE_UPDATE, isChecked).apply();
                    sendRuntimeConfig();
                }
            });
        }
        bootSwitch.setOnCheckedChangeListener(new CompoundButton.OnCheckedChangeListener() {
            @Override
            public void onCheckedChanged(CompoundButton buttonView, boolean isChecked) {
                prefs.edit().putBoolean(KEY_BOOT, isChecked).apply();
            }
        });
        watchdogSwitch.setOnCheckedChangeListener(new CompoundButton.OnCheckedChangeListener() {
            @Override
            public void onCheckedChanged(CompoundButton buttonView, boolean isChecked) {
                prefs.edit().putBoolean(KEY_WATCHDOG, isChecked).apply();
                sendRuntimeConfig();
            }
        });
        guardSwitch.setOnCheckedChangeListener(new CompoundButton.OnCheckedChangeListener() {
            @Override
            public void onCheckedChanged(CompoundButton buttonView, boolean isChecked) {
                prefs.edit().putBoolean(KEY_GUARD, isChecked).apply();
                sendRuntimeConfig();
            }
        });

        // 运行策略
        TextView rtTitle = new TextView(this);
        rtTitle.setText(R.string.settings_runtime_title);
        rtTitle.setTextSize(TypedValue.COMPLEX_UNIT_SP, 16);
        rtTitle.setTypeface(rtTitle.getTypeface(), android.graphics.Typeface.BOLD);
        rtTitle.setTextColor(UiTheme.title(this));
        LinearLayout.LayoutParams rtp = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.WRAP_CONTENT);
        rtp.topMargin = dp(18);
        root.addView(rtTitle, rtp);

        TextView rtHint = new TextView(this);
        rtHint.setText(R.string.settings_runtime_hint);
        rtHint.setTextSize(TypedValue.COMPLEX_UNIT_SP, 12);
        rtHint.setTextColor(UiTheme.hint(this));
        rtHint.setPadding(0, dp(4), 0, dp(8));
        root.addView(rtHint);

        Switch chargeSwitch = addSwitchRow(root, R.string.option_only_charging);
        Switch wifiSwitch = addSwitchRow(root, R.string.option_only_wifi);
        Switch dailySwitch = addSwitchRow(root, R.string.option_daily_restart);

        // 每天重启的时间（点击可修改）
        final Button timeButton = new Button(this);
        timeButton.setAllCaps(false);
        root.addView(timeButton, new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT, dp(44)));
        final int[] hm = {
                prefs.getInt(KEY_RESTART_HOUR, 4),
                prefs.getInt(KEY_RESTART_MIN, 0)
        };
        timeButton.setText(getString(R.string.option_restart_time, hm[0], hm[1]));
        timeButton.setEnabled(dailySwitch.isChecked());
        timeButton.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                new android.app.TimePickerDialog(SettingsActivity.this,
                        new android.app.TimePickerDialog.OnTimeSetListener() {
                            @Override
                            public void onTimeSet(android.widget.TimePicker view, int hourOfDay, int minute) {
                                hm[0] = hourOfDay;
                                hm[1] = minute;
                                prefs.edit()
                                        .putInt(KEY_RESTART_HOUR, hourOfDay)
                                        .putInt(KEY_RESTART_MIN, minute)
                                        .apply();
                                timeButton.setText(getString(R.string.option_restart_time, hourOfDay, minute));
                                sendRuntimeConfig();
                            }
                        }, hm[0], hm[1], true).show();
            }
        });

        chargeSwitch.setChecked(prefs.getBoolean(KEY_ONLY_CHARGING, false));
        wifiSwitch.setChecked(prefs.getBoolean(KEY_ONLY_WIFI, false));
        dailySwitch.setChecked(prefs.getBoolean(KEY_DAILY_RESTART, false));

        chargeSwitch.setOnCheckedChangeListener(new CompoundButton.OnCheckedChangeListener() {
            @Override
            public void onCheckedChanged(CompoundButton buttonView, boolean isChecked) {
                prefs.edit().putBoolean(KEY_ONLY_CHARGING, isChecked).apply();
                sendRuntimeConfig();
            }
        });
        wifiSwitch.setOnCheckedChangeListener(new CompoundButton.OnCheckedChangeListener() {
            @Override
            public void onCheckedChanged(CompoundButton buttonView, boolean isChecked) {
                prefs.edit().putBoolean(KEY_ONLY_WIFI, isChecked).apply();
                sendRuntimeConfig();
            }
        });
        dailySwitch.setOnCheckedChangeListener(new CompoundButton.OnCheckedChangeListener() {
            @Override
            public void onCheckedChanged(CompoundButton buttonView, boolean isChecked) {
                prefs.edit().putBoolean(KEY_DAILY_RESTART, isChecked).apply();
                timeButton.setEnabled(isChecked);
                sendRuntimeConfig();
            }
        });

        // 权限与系统设置
        TextView permTitle = new TextView(this);
        permTitle.setText(R.string.settings_perms_title);
        permTitle.setTextSize(TypedValue.COMPLEX_UNIT_SP, 16);
        permTitle.setTypeface(permTitle.getTypeface(), android.graphics.Typeface.BOLD);
        permTitle.setTextColor(UiTheme.title(this));
        LinearLayout.LayoutParams tp = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.WRAP_CONTENT);
        tp.topMargin = dp(18);
        root.addView(permTitle, tp);

        TextView permHint = new TextView(this);
        permHint.setText(R.string.settings_perms_hint);
        permHint.setTextSize(TypedValue.COMPLEX_UNIT_SP, 12);
        permHint.setTextColor(UiTheme.hint(this));
        permHint.setPadding(0, dp(4), 0, dp(8));
        root.addView(permHint);

        addActionButton(root, R.string.perm_notif, new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                android.app.NotificationManager nm =
                        (android.app.NotificationManager) getSystemService(NOTIFICATION_SERVICE);
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU
                        && checkSelfPermission(android.Manifest.permission.POST_NOTIFICATIONS)
                        != android.content.pm.PackageManager.PERMISSION_GRANTED) {
                    requestPermissions(new String[]{android.Manifest.permission.POST_NOTIFICATIONS}, 2);
                    return;
                }
                // 权限已给但通知总开关关闭（realme/OPPO 等）：跳到系统通知设置
                if (nm != null && !nm.areNotificationsEnabled()) {
                    try {
                        Intent intent = new Intent(Settings.ACTION_APP_NOTIFICATION_SETTINGS);
                        intent.putExtra(Settings.EXTRA_APP_PACKAGE, getPackageName());
                        startActivity(intent);
                    } catch (Exception ignored) {
                    }
                    return;
                }
                toast(R.string.perm_granted);
            }
        });
        addActionButton(root, R.string.perm_storage, new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                if (Build.VERSION.SDK_INT < Build.VERSION_CODES.R) {
                    // Android 6~10：旧式存储权限
                    if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M
                            && checkSelfPermission(android.Manifest.permission.WRITE_EXTERNAL_STORAGE)
                            != android.content.pm.PackageManager.PERMISSION_GRANTED) {
                        requestPermissions(new String[]{
                                android.Manifest.permission.READ_EXTERNAL_STORAGE,
                                android.Manifest.permission.WRITE_EXTERNAL_STORAGE}, 3);
                        return;
                    }
                    toast(R.string.perm_granted);
                    return;
                }
                if (Environment.isExternalStorageManager()) {
                    toast(R.string.perm_granted);
                    return;
                }
                prefs.edit().putBoolean("asked_storage", true).apply();
                try {
                    startActivity(new Intent(Settings.ACTION_MANAGE_APP_ALL_FILES_ACCESS_PERMISSION,
                            Uri.parse("package:" + getPackageName())));
                } catch (Exception e) {
                    try {
                        startActivity(new Intent(Settings.ACTION_MANAGE_ALL_FILES_ACCESS_PERMISSION));
                    } catch (Exception ignored) {
                    }
                }
            }
        });
        addActionButton(root, R.string.perm_battery, new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                PowerManager pm = (PowerManager) getSystemService(POWER_SERVICE);
                if (pm != null && pm.isIgnoringBatteryOptimizations(getPackageName())) {
                    toast(R.string.perm_granted);
                    return;
                }
                prefs.edit().putBoolean("asked_battery", true).apply();
                try {
                    startActivity(new Intent(Settings.ACTION_REQUEST_IGNORE_BATTERY_OPTIMIZATIONS,
                            Uri.parse("package:" + getPackageName())));
                } catch (Exception e) {
                    try {
                        startActivity(new Intent(Settings.ACTION_IGNORE_BATTERY_OPTIMIZATION_SETTINGS));
                    } catch (Exception ignored) {
                    }
                }
            }
        });
        addActionButton(root, R.string.perm_overlay, new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                if (Settings.canDrawOverlays(SettingsActivity.this)) {
                    toast(R.string.perm_granted);
                    return;
                }
                requestOverlayPermission();
            }
        });
        addActionButton(root, R.string.perm_autostart, new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                openOemPage(autostartComponents());
            }
        });
        addActionButton(root, R.string.perm_associate, new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                // 多数品牌的自启动管理页里同时包含关联启动开关
                openOemPage(autostartComponents());
            }
        });

        Button back = new Button(this);
        back.setText(R.string.settings_back);
        back.setOnClickListener(new View.OnClickListener() {
            @Override
            public void onClick(View v) {
                finish();
            }
        });
        LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.WRAP_CONTENT);
        lp.topMargin = dp(16);
        root.addView(back, lp);

        setContentView(scroll);
        applySystemInsets(root);
    }

    /** 按系统栏高度补内边距，避免内容画到状态栏/导航栏下面。 */
    private void applySystemInsets(final View root) {
        final int l = root.getPaddingLeft();
        final int t = root.getPaddingTop();
        final int r = root.getPaddingRight();
        final int b = root.getPaddingBottom();
        root.setOnApplyWindowInsetsListener(new View.OnApplyWindowInsetsListener() {
            @Override
            public android.view.WindowInsets onApplyWindowInsets(View v, android.view.WindowInsets insets) {
                int top;
                int bottom;
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.R) {
                    android.graphics.Insets bars = insets.getInsets(android.view.WindowInsets.Type.systemBars());
                    top = bars.top;
                    bottom = bars.bottom;
                } else {
                    top = insets.getSystemWindowInsetTop();
                    bottom = insets.getSystemWindowInsetBottom();
                }
                v.setPadding(l, t + top, r, b + bottom);
                return insets;
            }
        });
        root.requestApplyInsets();
    }

    private Switch addSwitchRow(LinearLayout parent, int labelRes) {
        LinearLayout row = new LinearLayout(this);
        row.setOrientation(LinearLayout.HORIZONTAL);
        row.setPadding(0, dp(4), 0, dp(4));
        TextView label = new TextView(this);
        label.setText(labelRes);
        label.setTextColor(UiTheme.title(this));
        label.setTextSize(TypedValue.COMPLEX_UNIT_SP, 14);
        row.addView(label, new LinearLayout.LayoutParams(0,
                ViewGroup.LayoutParams.WRAP_CONTENT, 1));
        Switch toggle = new Switch(this);
        row.addView(toggle, new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.WRAP_CONTENT,
                ViewGroup.LayoutParams.WRAP_CONTENT));
        parent.addView(row);
        return toggle;
    }

    private void applyScreenOn(boolean on) {
        if (on) {
            getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);
        } else {
            getWindow().clearFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);
        }
    }

    private void sendOverlayConfig() {
        Intent intent = new Intent(this, BotService.class);
        intent.setAction(BotService.ACTION_OVERLAY);
        intent.putExtra(BotService.EXTRA_OVERLAY_ON, overlaySwitch.isChecked());
        intent.putExtra(BotService.EXTRA_OVERLAY_OPAQUE, opaqueSwitch.isChecked());
        intent.putExtra(BotService.EXTRA_OVERLAY_CONTENT, currentContent());
        startService(intent);
    }

    /** 已保存的悬浮窗显示内容（兼容旧值 both）。 */
    private String currentContent() {
        String value = prefs.getString(KEY_OVERLAY_CONTENT, "status,uptime");
        if ("both".equals(value) || value == null || value.length() == 0) {
            return "status,uptime";
        }
        return value;
    }

    private String themeLabel(String mode) {
        if (UiTheme.LIGHT.equals(mode)) {
            return getString(R.string.theme_light);
        }
        if (UiTheme.DARK.equals(mode)) {
            return getString(R.string.theme_dark);
        }
        return getString(R.string.theme_system);
    }

    private String contentLabel(String value) {
        StringBuilder sb = new StringBuilder();
        if (value == null || value.length() == 0 || "both".equals(value)) {
            value = "status,uptime";
        }
        if (value.contains("status")) {
            sb.append(getString(R.string.overlay_content_status));
        }
        if (value.contains("uptime")) {
            if (sb.length() > 0) {
                sb.append(" / ");
            }
            sb.append(getString(R.string.overlay_content_uptime));
        }
        if (value.contains("port")) {
            if (sb.length() > 0) {
                sb.append(" / ");
            }
            sb.append(getString(R.string.overlay_content_port));
        }
        if (value.contains("accounts")) {
            if (sb.length() > 0) {
                sb.append(" / ");
            }
            sb.append(getString(R.string.overlay_content_accounts));
        }
        if (value.contains("stats")) {
            if (sb.length() > 0) {
                sb.append(" / ");
            }
            sb.append(getString(R.string.overlay_content_stats));
        }
        return sb.length() > 0 ? sb.toString() : getString(R.string.overlay_content_status);
    }

    private void sendRuntimeConfig() {
        Intent intent = new Intent(this, BotService.class);
        intent.setAction(BotService.ACTION_CONFIGURE);
        startService(intent);
    }

    private void requestOverlayPermission() {
        try {
            startActivity(new Intent(Settings.ACTION_MANAGE_OVERLAY_PERMISSION,
                    Uri.parse("package:" + getPackageName())));
        } catch (Exception e) {
            try {
                startActivity(new Intent(Settings.ACTION_MANAGE_OVERLAY_PERMISSION));
            } catch (Exception ignored) {
            }
        }
    }

    private void addActionButton(LinearLayout parent, int textRes, View.OnClickListener l) {
        Button button = new Button(this);
        button.setText(textRes);
        button.setAllCaps(false);
        button.setOnClickListener(l);
        LinearLayout.LayoutParams lp = new LinearLayout.LayoutParams(
                ViewGroup.LayoutParams.MATCH_PARENT,
                ViewGroup.LayoutParams.WRAP_CONTENT);
        lp.topMargin = dp(8);
        parent.addView(button, lp);
    }

    private void toast(int res) {
        Toast.makeText(this, res, Toast.LENGTH_SHORT).show();
    }

    /** 按厂商给出候选的自启动/关联启动管理页，按顺序尝试。 */
    private String[] autostartComponents() {
        String m = (Build.MANUFACTURER + " " + Build.BRAND).toLowerCase(java.util.Locale.ROOT);
        if (m.contains("xiaomi") || m.contains("redmi") || m.contains("poco")) {
            return new String[]{
                    "com.miui.securitycenter/com.miui.permcenter.autostart.AutoStartManagementActivity"};
        }
        if (m.contains("huawei") || m.contains("honor")) {
            return new String[]{
                    "com.huawei.systemmanager/.startupmgr.ui.StartupNormalAppListActivity",
                    "com.huawei.systemmanager/.optimize.process.ProtectActivity"};
        }
        if (m.contains("oppo") || m.contains("realme") || m.contains("oneplus")) {
            return new String[]{
                    "com.coloros.safecenter/.permission.startup.StartupAppListActivity",
                    "com.oppo.safe/.permission.startup.StartupAppListActivity",
                    "com.coloros.safecenter/.startupapp.StartupAppListActivity"};
        }
        if (m.contains("vivo") || m.contains("iqoo")) {
            return new String[]{
                    "com.vivo.permissionmanager/.activity.BgStartUpManagerActivity",
                    "com.iqoo.secure/.ui.PhoneOptimize.AddWhiteListActivity"};
        }
        if (m.contains("meizu")) {
            return new String[]{
                    "com.meizu.safe/.permission.SmartBGActivity",
                    "com.meizu.safe/.security.SSMStartmanager"};
        }
        if (m.contains("samsung")) {
            return new String[]{
                    "com.samsung.android.lool/.activity.SleepingAppsActivity",
                    "com.samsung.android.lool/.activity.BatteryActivity"};
        }
        // 未知品牌：依次尝试各家常见入口
        return new String[]{
                "com.miui.securitycenter/com.miui.permcenter.autostart.AutoStartManagementActivity",
                "com.vivo.permissionmanager/.activity.BgStartUpManagerActivity",
                "com.coloros.safecenter/.permission.startup.StartupAppListActivity",
                "com.huawei.systemmanager/.startupmgr.ui.StartupNormalAppListActivity"};
    }

    /** 跳转国产 ROM 的自启动/关联启动管理页；找不到就退到应用详情页。 */
    private void openOemPage(String... components) {
        for (String c : components) {
            String[] parts = c.split("/");
            try {
                Intent intent = new Intent();
                intent.setComponent(new ComponentName(parts[0], parts[1]));
                intent.addFlags(Intent.FLAG_ACTIVITY_NEW_TASK);
                startActivity(intent);
                return;
            } catch (Exception ignored) {
            }
        }
        toast(R.string.perm_manual_hint);
        try {
            startActivity(new Intent(Settings.ACTION_APPLICATION_DETAILS_SETTINGS,
                    Uri.parse("package:" + getPackageName())));
        } catch (Exception ignored) {
        }
    }

    private int dp(int v) {
        return Math.round(getResources().getDisplayMetrics().density * v);
    }
}
