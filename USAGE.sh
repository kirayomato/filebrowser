irm https://get.pnpm.io/ps1 | iex

build-android-arm64.bat
adb push dist\filebrowser-android-arm64 /data/local/tmp/
adb forward tcp:8888 tcp:8888

adb shell
cd /data/local/tmp/
chmod +x ./filebrowser-android-arm64
# 后台常驻启动
setsid ./filebrowser-android-arm64 -p 8888 -a 0.0.0.0 -r /storage/emulated/0/ > filebrowser_log.txt 2>&1 &

# http://localhost:8888/files/Android/data/com.tencent.mobileqq/Tencent/MobileQQ/chatpic

