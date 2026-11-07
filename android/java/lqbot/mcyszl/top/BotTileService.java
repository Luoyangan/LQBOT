package lqbot.mcyszl.top;

import android.content.Intent;
import android.os.Build;
import android.service.quicksettings.Tile;
import android.service.quicksettings.TileService;

/**
 * 下拉快捷开关：一键启动/停止内核，状态与内核实际运行状态同步。
 * 仅在 Android 7.0（API 24）及以上由系统绑定。
 */
public class BotTileService extends TileService {
    @Override
    public void onStartListening() {
        super.onStartListening();
        updateTile();
    }

    @Override
    public void onClick() {
        super.onClick();
        String action = BotService.isKernelStarted()
                ? BotService.ACTION_STOP : BotService.ACTION_START;
        Intent intent = new Intent(this, BotService.class);
        intent.setAction(action);
        try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                startForegroundService(intent);
            } else {
                startService(intent);
            }
        } catch (Exception ignored) {
        }
        // 状态由服务在几百毫秒内切换，稍后刷新即可
        updateTile();
    }

    private void updateTile() {
        Tile tile = getQsTile();
        if (tile == null) {
            return;
        }
        boolean running = BotService.isKernelStarted();
        tile.setState(running ? Tile.STATE_ACTIVE : Tile.STATE_INACTIVE);
        tile.setLabel(getString(R.string.tile_label));
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            tile.setSubtitle(getString(running ? R.string.status_running : R.string.status_stopped));
        }
        tile.updateTile();
    }
}
