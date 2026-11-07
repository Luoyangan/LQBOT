package lqbot.mcyszl.top;

import android.content.BroadcastReceiver;
import android.content.Context;
import android.content.Intent;

/** 定时看门狗（约每 15 分钟）：进程被系统清掉后，闹钟仍能把服务重新拉起。 */
public class WatchdogReceiver extends BroadcastReceiver {
    @Override
    public void onReceive(Context context, Intent intent) {
        String action = intent == null ? null : intent.getAction();
        if (BotService.ACTION_DAILY_RESTART.equals(action)) {
            // 每天定时重启：关闭该功能后忽略残留闹钟
            boolean daily = context.getSharedPreferences("lqbot_prefs", Context.MODE_PRIVATE)
                    .getBoolean("daily_restart", false);
            if (!daily) {
                return;
            }
            BootReceiver.start(context, BotService.class, BotService.ACTION_RESTART);
            BootReceiver.start(context, GuardService.class, GuardService.ACTION_START);
            return;
        }
        // 控制台关闭「定时看门狗」后忽略残留闹钟
        boolean enabled = context.getSharedPreferences("lqbot_prefs", Context.MODE_PRIVATE)
                .getBoolean("watchdog", true);
        if (!enabled) {
            return;
        }
        BootReceiver.start(context, BotService.class, BotService.ACTION_START);
        BootReceiver.start(context, GuardService.class, GuardService.ACTION_START);
    }
}
