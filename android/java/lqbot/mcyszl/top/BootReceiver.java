package lqbot.mcyszl.top;

import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;
import android.os.Build;

/** 开机自启：系统启动完成后拉起内核服务与守护进程。 */
public class BootReceiver extends BroadcastReceiver {
    @Override
    public void onReceive(Context context, Intent intent) {
        if (!Intent.ACTION_BOOT_COMPLETED.equals(intent.getAction())
                && !"android.intent.action.MY_PACKAGE_REPLACED".equals(intent.getAction())) {
            return;
        }
        // 控制台关闭「开机自启」后不再拉起
        boolean enabled = context.getSharedPreferences("lqbot_prefs", Context.MODE_PRIVATE)
                .getBoolean("boot_autostart", true);
        if (!enabled) {
            return;
        }
        start(context, BotService.class, BotService.ACTION_START);
        start(context, GuardService.class, GuardService.ACTION_START);
    }

    static void start(Context context, Class<?> service, String action) {
        Intent intent = new Intent(context, service);
        intent.setAction(action);
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            context.startForegroundService(intent);
        } else {
            context.startService(intent);
        }
    }
}
