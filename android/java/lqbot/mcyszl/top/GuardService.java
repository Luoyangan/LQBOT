package lqbot.mcyszl.top;

import android.app.Notification;
import android.app.NotificationChannel;
import android.app.NotificationManager;
import android.app.PendingIntent;
import android.app.Service;
import android.content.Intent;
import android.os.Build;
import android.os.Handler;
import android.os.IBinder;
import android.os.Looper;
import android.os.PowerManager;
import android.util.Log;

import java.net.InetSocketAddress;
import java.net.Socket;

/**
 * 独立进程（:guard）的后台守护：每 30 秒探测本机管理后台端口，连续 3 次不可达
 * 就重新拉起 BotService。与 BotService 互相拉起，降低被国产 ROM 一并清掉后无法恢复的概率。
 */
public class GuardService extends Service {
    public static final String ACTION_START = "com.lqbot.app.GUARD_START";

    private static final String TAG = "LQBOT";
    private static final String CHANNEL_ID = "lqbot_guard";
    private static final int NOTIF_ID = 1002;
    private static final int PORT = 4789;
    private static final long CHECK_MS = 15_000L;
    private static final int FAILS_TO_RESTART = 2;

    private final Handler handler = new Handler(Looper.getMainLooper());
    private PowerManager.WakeLock wakeLock;
    private int fails;

    private final Runnable check = new Runnable() {
        @Override
        public void run() {
            if (isPortUp()) {
                fails = 0;
            } else if (++fails >= FAILS_TO_RESTART) {
                fails = 0;
                Log.i(TAG, "[守护] 检测到内核不在线，重新拉起服务");
                startBot();
            }
            handler.postDelayed(this, CHECK_MS);
        }
    };

    @Override
    public void onCreate() {
        super.onCreate();
        createChannel();
        acquireWakeLock();
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        startForeground(NOTIF_ID, buildNotification());
        handler.removeCallbacks(check);
        handler.postDelayed(check, CHECK_MS);
        return START_STICKY;
    }

    @Override
    public IBinder onBind(Intent intent) {
        return null;
    }

    @Override
    public void onDestroy() {
        handler.removeCallbacks(check);
        if (wakeLock != null && wakeLock.isHeld()) {
            wakeLock.release();
        }
        wakeLock = null;
        super.onDestroy();
    }

    private void createChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) {
            return;
        }
        NotificationChannel channel = new NotificationChannel(
                CHANNEL_ID,
                getString(R.string.guard_channel_name),
                NotificationManager.IMPORTANCE_MIN);
        channel.setDescription(getString(R.string.guard_channel_desc));
        NotificationManager nm = (NotificationManager) getSystemService(NOTIFICATION_SERVICE);
        if (nm != null) {
            nm.createNotificationChannel(channel);
        }
    }

    private Notification buildNotification() {
        Intent open = new Intent(this, MainActivity.class);
        open.setFlags(Intent.FLAG_ACTIVITY_SINGLE_TOP);
        PendingIntent contentIntent = PendingIntent.getActivity(this, 0, open,
                PendingIntent.FLAG_IMMUTABLE);
        Notification.Builder builder;
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            builder = new Notification.Builder(this, CHANNEL_ID);
        } else {
            builder = new Notification.Builder(this);
        }
        return builder
                .setContentTitle(getString(R.string.app_name))
                .setContentText(getString(R.string.notif_guard))
                .setSmallIcon(android.R.drawable.ic_menu_compass)
                .setContentIntent(contentIntent)
                .setOngoing(true)
                .build();
    }

    private void acquireWakeLock() {
        PowerManager pm = (PowerManager) getSystemService(POWER_SERVICE);
        if (pm == null) {
            return;
        }
        wakeLock = pm.newWakeLock(PowerManager.PARTIAL_WAKE_LOCK, "lqbot:guard");
        wakeLock.setReferenceCounted(false);
        wakeLock.acquire();
    }

    private void startBot() {
        Intent intent = new Intent(this, BotService.class);
        intent.setAction(BotService.ACTION_START);
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            startForegroundService(intent);
        } else {
            startService(intent);
        }
    }

    private boolean isPortUp() {
        Socket socket = new Socket();
        try {
            socket.connect(new InetSocketAddress("127.0.0.1", PORT), 500);
            return true;
        } catch (Exception e) {
            return false;
        } finally {
            try {
                socket.close();
            } catch (Exception ignored) {
            }
        }
    }
}
