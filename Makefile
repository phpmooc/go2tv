VERSION=$(shell cat version.txt)
LDFLAGS="-s -w -X main.version=$(VERSION)"
LDFLAGS_WINDOWS="-s -w -H=windowsgui -X main.version=$(VERSION)"
TAGS?=migrated_fynedo

BUILD_DIR=build
BIN=$(BUILD_DIR)/go2tv
BIN_WIN=$(BUILD_DIR)/go2tv.exe
APPDIR=$(BUILD_DIR)/AppDir
DESKTOP_SRC=assets/linux/app.go2tv.go2tv.desktop
DESKTOP_APPDIR=$(APPDIR)/usr/share/applications/app.go2tv.go2tv.desktop
DESKTOP_ROOT=$(APPDIR)/app.go2tv.go2tv.desktop
ICON_SRC=assets/go2tv-icon-desktop-512.png
ICON_APPDIR=$(APPDIR)/usr/share/icons/hicolor/512x512/apps/go2tv.png
ICON_ROOT=$(APPDIR)/go2tv.png
APPDATA_SRC=assets/linux/app.go2tv.go2tv.appdata.xml
APPDATA_APPDIR=$(APPDIR)/usr/share/metainfo
APPIMAGETOOL=$(BUILD_DIR)/appimagetool
ARCH:=$(shell uname -m)
APPIMAGE_OUT=$(BUILD_DIR)/Go2TV-$(ARCH).AppImage
FFMPEG_STATIC_ARCHIVE=$(BUILD_DIR)/ffmpeg-static.tar.xz
FFMPEG_STATIC_DIR=$(BUILD_DIR)/ffmpeg-static
FFMPEG_APP_LIBDIR=$(APPDIR)/usr/lib/ffmpeg
APPIMAGE_FFMPEG_MODE?=auto
FYNE?=fyne
APK_OUT=$(BUILD_DIR)/Go2TV.apk
APK_FFMPEG_OUT=$(BUILD_DIR)/Go2TV-ffmpeg.apk
APK_FFMPEG_ALIGNED=$(BUILD_DIR)/Go2TV-ffmpeg-aligned.apk
ANDROID_FFMPEG_BASE_URL?=https://raw.githubusercontent.com/hzw1199/Android-FFmpeg-Prebuilt/main/ffmpeg-8.1.1/bin
ANDROID_FFMPEG_URL?=$(ANDROID_FFMPEG_BASE_URL)/ffmpeg
ANDROID_FFPROBE_URL?=$(ANDROID_FFMPEG_BASE_URL)/ffprobe
ANDROID_FFMPEG_BIN=$(BUILD_DIR)/ffmpeg-android
ANDROID_FFPROBE_BIN=$(BUILD_DIR)/ffprobe-android
ANDROID_APK_LIBS=$(BUILD_DIR)/apk-libs
ANDROID_ABI?=arm64-v8a
ANDROID_BUILD_TOOLS?=$(shell ls -d $$ANDROID_HOME/build-tools/* 2>/dev/null | sort -V | tail -n1)

.PHONY: build wayland windows install uninstall clean run test appimage appimage-ffmpeg android android-ffmpeg

build: clean
	go build -tags "$(TAGS)" -trimpath -ldflags $(LDFLAGS) -o $(BIN) ./cmd/go2tv

wayland: clean
	go build -tags "$(TAGS),wayland" -trimpath -ldflags $(LDFLAGS) -o $(BIN) ./cmd/go2tv

windows: clean
	env CGO_ENABLED=1 CC=$$(command -v x86_64-w64-mingw32-gcc-win32 || echo x86_64-w64-mingw32-gcc) CXX=$$(command -v x86_64-w64-mingw32-g++-win32 || echo x86_64-w64-mingw32-g++) CGO_LDFLAGS="-static -static-libgcc -static-libstdc++ -Wl,-Bstatic -l:libstdc++.a -l:libwinpthread.a" GOOS=windows GOARCH=amd64 go build -tags "$(TAGS)" -trimpath -ldflags "$(LDFLAGS_WINDOWS) -linkmode external -extldflags '-static'" -o $(BIN_WIN) ./cmd/go2tv


install: build
	mkdir -vp /usr/local/bin/
	cp $(BIN) /usr/local/bin/
	$(MAKE) clean

uninstall:
	rm -vf /usr/local/bin/go2tv

clean:
	rm -rf $(BUILD_DIR)

run: build
	$(BIN)

test:
	go test -v ./...

android:
	set -e; \
	if [ -z "$$ANDROID_NDK_HOME" ]; then echo "ANDROID_NDK_HOME is required"; exit 1; fi; \
	mkdir -p $(BUILD_DIR); \
	rm -f $(APK_OUT); \
	cd cmd/go2tv; \
	rm -f ./*.apk; \
	ANDROID_NDK_HOME="$$ANDROID_NDK_HOME" $(FYNE) package \
		--os android \
		--name Go2TV \
		--app-id app.go2tv.go2tv \
		--icon ../../assets/go2tv-icon-android.png \
		--app-version "$(VERSION)" \
		--app-build 1 \
		--release; \
	APK_BUILT="$$(find . -maxdepth 1 -type f -name '*.apk' | head -n 1)"; \
	if [ -z "$$APK_BUILT" ]; then echo "fyne did not create an APK"; exit 1; fi; \
	mv "$$APK_BUILT" ../../$(APK_OUT); \
	echo "APK created at $(APK_OUT)"

android-ffmpeg:
	set -e; \
	if [ -z "$$ANDROID_NDK_HOME" ]; then echo "ANDROID_NDK_HOME is required"; exit 1; fi; \
	if [ -z "$$ANDROID_HOME" ]; then echo "ANDROID_HOME is required"; exit 1; fi; \
	if [ -z "$(ANDROID_BUILD_TOOLS)" ]; then echo "Android build-tools not found under ANDROID_HOME"; exit 1; fi; \
	if [ ! -x "$(ANDROID_BUILD_TOOLS)/aapt" ]; then echo "aapt missing in $(ANDROID_BUILD_TOOLS)"; exit 1; fi; \
	if [ ! -x "$(ANDROID_BUILD_TOOLS)/zipalign" ]; then echo "zipalign missing in $(ANDROID_BUILD_TOOLS)"; exit 1; fi; \
	if [ ! -x "$(ANDROID_BUILD_TOOLS)/apksigner" ]; then echo "apksigner missing in $(ANDROID_BUILD_TOOLS)"; exit 1; fi; \
	if ! command -v zip >/dev/null 2>&1; then echo "zip is required"; exit 1; fi; \
	if ! command -v keytool >/dev/null 2>&1; then echo "keytool is required"; exit 1; fi; \
	mkdir -p $(BUILD_DIR); \
	rm -rf $(ANDROID_APK_LIBS) $(ANDROID_FFMPEG_BIN) $(ANDROID_FFPROBE_BIN) $(APK_FFMPEG_OUT) $(APK_FFMPEG_ALIGNED); \
	cd cmd/go2tv; \
	rm -f ./*.apk; \
	ANDROID_NDK_HOME="$$ANDROID_NDK_HOME" $(FYNE) package \
		--os android/arm64 \
		--name "Go2TV ffmpeg" \
		--app-id app.go2tv.go2tv \
		--icon ../../assets/go2tv-icon-android.png \
		--app-version "$(VERSION)" \
		--app-build 1 \
		--release; \
	APK_BUILT="$$(find . -maxdepth 1 -type f -name '*.apk' | head -n 1)"; \
	if [ -z "$$APK_BUILT" ]; then echo "fyne did not create an APK"; exit 1; fi; \
	mv "$$APK_BUILT" ../../$(APK_FFMPEG_OUT); \
	cd ../..; \
	echo "Downloading android ffmpeg: $(ANDROID_FFMPEG_URL)"; \
	curl -fsSL "$(ANDROID_FFMPEG_URL)" -o $(ANDROID_FFMPEG_BIN) || wget -q -O $(ANDROID_FFMPEG_BIN) "$(ANDROID_FFMPEG_URL)"; \
	echo "Downloading android ffprobe: $(ANDROID_FFPROBE_URL)"; \
	curl -fsSL "$(ANDROID_FFPROBE_URL)" -o $(ANDROID_FFPROBE_BIN) || wget -q -O $(ANDROID_FFPROBE_BIN) "$(ANDROID_FFPROBE_URL)"; \
	chmod 755 $(ANDROID_FFMPEG_BIN) $(ANDROID_FFPROBE_BIN); \
	mkdir -p $(ANDROID_APK_LIBS)/lib/$(ANDROID_ABI); \
	cp $(ANDROID_FFMPEG_BIN) $(ANDROID_APK_LIBS)/lib/$(ANDROID_ABI)/libffmpeg.so; \
	cp $(ANDROID_FFPROBE_BIN) $(ANDROID_APK_LIBS)/lib/$(ANDROID_ABI)/libffprobe.so; \
	chmod 755 $(ANDROID_APK_LIBS)/lib/$(ANDROID_ABI)/libffmpeg.so $(ANDROID_APK_LIBS)/lib/$(ANDROID_ABI)/libffprobe.so; \
	( cd $(ANDROID_APK_LIBS) && zip -q -g ../$(notdir $(APK_FFMPEG_OUT)) lib/$(ANDROID_ABI)/libffmpeg.so lib/$(ANDROID_ABI)/libffprobe.so ); \
	MANIFEST_DUMP="$$($(ANDROID_BUILD_TOOLS)/aapt dump xmltree $(APK_FFMPEG_OUT) AndroidManifest.xml || true)"; \
	if echo "$$MANIFEST_DUMP" | grep -E "extractNativeLibs.*(false|0x0)" >/dev/null; then \
		echo "AndroidManifest sets extractNativeLibs=false"; \
		exit 1; \
	fi; \
	$(ANDROID_BUILD_TOOLS)/zipalign -f 4 $(APK_FFMPEG_OUT) $(APK_FFMPEG_ALIGNED); \
	mv $(APK_FFMPEG_ALIGNED) $(APK_FFMPEG_OUT); \
	if [ -n "$${GO2TV_ANDROID_KEYSTORE:-}" ] && [ -z "$${GO2TV_ANDROID_KEYSTORE_PASS:-}" ]; then echo "GO2TV_ANDROID_KEYSTORE_PASS is required with GO2TV_ANDROID_KEYSTORE"; exit 1; fi; \
	KEYSTORE="$${GO2TV_ANDROID_KEYSTORE:-$(BUILD_DIR)/go2tv-debug.keystore}"; \
	KEY_ALIAS="$${GO2TV_ANDROID_KEY_ALIAS:-go2tv}"; \
	STOREPASS="$${GO2TV_ANDROID_KEYSTORE_PASS:-android}"; \
	KEYPASS="$${GO2TV_ANDROID_KEY_PASS:-$$STOREPASS}"; \
	if [ -z "$${GO2TV_ANDROID_KEYSTORE:-}" ] && [ ! -f "$$KEYSTORE" ]; then \
		keytool -genkeypair -v -keystore "$$KEYSTORE" -storepass "$$STOREPASS" -keypass "$$KEYPASS" -alias "$$KEY_ALIAS" -keyalg RSA -keysize 2048 -validity 10000 -dname "CN=Go2TV,O=Go2TV,C=US"; \
	fi; \
	$(ANDROID_BUILD_TOOLS)/apksigner sign --ks "$$KEYSTORE" --ks-key-alias "$$KEY_ALIAS" --ks-pass pass:"$$STOREPASS" --key-pass pass:"$$KEYPASS" $(APK_FFMPEG_OUT); \
	$(ANDROID_BUILD_TOOLS)/apksigner verify $(APK_FFMPEG_OUT); \
	rm -rf $(ANDROID_APK_LIBS) $(ANDROID_FFMPEG_BIN) $(ANDROID_FFPROBE_BIN); \
	echo "APK created at $(APK_FFMPEG_OUT)"

appimage: build
	# Prepare AppDir structure
	rm -rf $(APPDIR)
	mkdir -p $(APPDIR)/usr/bin
	mkdir -p $(APPDIR)/usr/share/applications
	mkdir -p $(APPDIR)/usr/share/icons/hicolor/512x512/apps

	# Copy binary
	cp $(BIN) $(APPDIR)/usr/bin/

	# Generate minimal AppRun launcher
	printf '#!/bin/sh\nAPPDIR="$${APPDIR:-$$PWD}"\nexec "$$APPDIR/usr/bin/go2tv" "$$@"\n' > $(APPDIR)/AppRun
	chmod +x $(APPDIR)/AppRun

	# Desktop entry and icon
	# Use provided desktop file and 512x512 icon
	cp $(DESKTOP_SRC) $(DESKTOP_APPDIR)
	cp $(ICON_SRC) $(ICON_APPDIR)
	# Also place a desktop file and icon at AppDir root as required by appimagetool
	cp $(DESKTOP_SRC) $(DESKTOP_ROOT)
	cp $(ICON_SRC) $(ICON_ROOT)

	# AppStream metadata (removes appimagetool warning)
	mkdir -p $(APPDATA_APPDIR)
	cp $(APPDATA_SRC) $(APPDATA_APPDIR)/

	# Ensure Exec and Icon fields are correct inside the desktop file
	sed -i 's/^Exec=.*/Exec=go2tv/g; s/^Icon=.*/Icon=go2tv/g' $(DESKTOP_APPDIR)
	sed -i 's/^Exec=.*/Exec=go2tv/g; s/^Icon=.*/Icon=go2tv/g' $(DESKTOP_ROOT)

	# Fetch appimagetool if missing
	if [ ! -x $(APPIMAGETOOL) ]; then \
		URL="https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-$(ARCH).AppImage"; \
		curl -L "$$URL" -o $(APPIMAGETOOL) || wget -O $(APPIMAGETOOL) "$$URL"; \
		chmod +x $(APPIMAGETOOL); \
	fi

	# Build the AppImage
	( cd $(BUILD_DIR) && ./appimagetool AppDir "$(notdir $(APPIMAGE_OUT))" ); \
	echo "AppImage created at $(APPIMAGE_OUT)"

appimage-ffmpeg: build
	# Prepare AppDir structure
	rm -rf $(APPDIR)
	mkdir -p $(APPDIR)/usr/bin
	mkdir -p $(APPDIR)/usr/lib
	mkdir -p $(APPDIR)/usr/share/applications
	mkdir -p $(APPDIR)/usr/share/icons/hicolor/512x512/apps

	# Copy binary
	cp $(BIN) $(APPDIR)/usr/bin/

	# Bundle ffmpeg/ffprobe (modes: auto, system, download, none)
	set -e; \
	FFMPEG_MODE="$(APPIMAGE_FFMPEG_MODE)"; \
	FFMPEG_BIN="$${APPIMAGE_FFMPEG_BIN:-}"; \
	FFPROBE_BIN="$${APPIMAGE_FFPROBE_BIN:-}"; \
	if [ "$$FFMPEG_MODE" != "none" ]; then \
		if [ -z "$$FFMPEG_BIN" ] || [ -z "$$FFPROBE_BIN" ]; then \
			if [ "$$FFMPEG_MODE" = "auto" ] || [ "$$FFMPEG_MODE" = "system" ]; then \
				FFMPEG_BIN="$${FFMPEG_BIN:-$$(command -v ffmpeg || true)}"; \
				FFPROBE_BIN="$${FFPROBE_BIN:-$$(command -v ffprobe || true)}"; \
			fi; \
		fi; \
		if [ "$$FFMPEG_MODE" = "auto" ] && [ -n "$$FFMPEG_BIN" ] && [ -n "$$FFPROBE_BIN" ] && command -v ldd >/dev/null 2>&1; then \
			if ! ldd "$$FFMPEG_BIN" 2>/dev/null | grep -q "not a dynamic executable"; then \
				echo "Host ffmpeg is dynamic; switching to bundled ffmpeg for AppImage portability"; \
				FFMPEG_BIN=""; \
				FFPROBE_BIN=""; \
			fi; \
		fi; \
		if [ -z "$$FFMPEG_BIN" ] || [ -z "$$FFPROBE_BIN" ]; then \
			if [ "$$FFMPEG_MODE" = "system" ]; then \
				echo "APPIMAGE_FFMPEG_MODE=system but ffmpeg/ffprobe not found"; \
				exit 1; \
			fi; \
			case "$(ARCH)" in \
				x86_64) FFMPEG_URL="https://github.com/yt-dlp/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-linux64-gpl.tar.xz" ;; \
				aarch64|arm64) FFMPEG_URL="https://github.com/yt-dlp/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-linuxarm64-gpl.tar.xz" ;; \
				armv7l|armhf) FFMPEG_URL="https://github.com/yt-dlp/FFmpeg-Builds/releases/download/latest/ffmpeg-master-latest-linuxarmhf-gpl.tar.xz" ;; \
				*) echo "Unsupported arch for auto ffmpeg download: $(ARCH)"; exit 1 ;; \
			esac; \
			FFMPEG_URL="$${APPIMAGE_FFMPEG_URL:-$$FFMPEG_URL}"; \
			rm -rf $(FFMPEG_STATIC_DIR) $(FFMPEG_STATIC_ARCHIVE); \
			echo "Downloading ffmpeg bundle: $$FFMPEG_URL"; \
			curl -fsSL "$$FFMPEG_URL" -o $(FFMPEG_STATIC_ARCHIVE) || wget -q -O $(FFMPEG_STATIC_ARCHIVE) "$$FFMPEG_URL"; \
			mkdir -p $(FFMPEG_STATIC_DIR); \
			tar -xf $(FFMPEG_STATIC_ARCHIVE) -C $(FFMPEG_STATIC_DIR); \
			FFMPEG_BIN="$$(find $(FFMPEG_STATIC_DIR) -type f -name ffmpeg | head -n 1)"; \
			FFPROBE_BIN="$$(find $(FFMPEG_STATIC_DIR) -type f -name ffprobe | head -n 1)"; \
		fi; \
		if [ -z "$$FFMPEG_BIN" ] || [ -z "$$FFPROBE_BIN" ]; then \
			echo "Failed to resolve ffmpeg/ffprobe binaries for AppImage"; \
			exit 1; \
		fi; \
		cp "$$FFMPEG_BIN" $(APPDIR)/usr/bin/ffmpeg; \
		cp "$$FFPROBE_BIN" $(APPDIR)/usr/bin/ffprobe; \
		FFMPEG_LIB_HINT="$$(dirname "$$FFMPEG_BIN")/../lib"; \
		if [ -d "$$FFMPEG_LIB_HINT" ]; then \
			mkdir -p $(FFMPEG_APP_LIBDIR); \
			cp -a "$$FFMPEG_LIB_HINT"/. $(FFMPEG_APP_LIBDIR)/; \
		fi; \
		if command -v ldd >/dev/null 2>&1; then \
			if LD_LIBRARY_PATH="$(FFMPEG_APP_LIBDIR):$$LD_LIBRARY_PATH" ldd $(APPDIR)/usr/bin/ffmpeg 2>/dev/null | grep -q "not found"; then \
				echo "Unresolved shared libs for bundled ffmpeg:"; \
				LD_LIBRARY_PATH="$(FFMPEG_APP_LIBDIR):$$LD_LIBRARY_PATH" ldd $(APPDIR)/usr/bin/ffmpeg 2>/dev/null | grep "not found" || true; \
				exit 1; \
			fi; \
			if LD_LIBRARY_PATH="$(FFMPEG_APP_LIBDIR):$$LD_LIBRARY_PATH" ldd $(APPDIR)/usr/bin/ffprobe 2>/dev/null | grep -q "not found"; then \
				echo "Unresolved shared libs for bundled ffprobe:"; \
				LD_LIBRARY_PATH="$(FFMPEG_APP_LIBDIR):$$LD_LIBRARY_PATH" ldd $(APPDIR)/usr/bin/ffprobe 2>/dev/null | grep "not found" || true; \
				exit 1; \
			fi; \
		fi; \
	fi

	# Generate minimal AppRun launcher
	printf '#!/bin/sh\nAPPDIR="$${APPDIR:-$$PWD}"\nexport PATH="$$APPDIR/usr/bin:$$PATH"\nexport LD_LIBRARY_PATH="$$APPDIR/usr/lib/ffmpeg:$$APPDIR/usr/lib:$$LD_LIBRARY_PATH"\nexec "$$APPDIR/usr/bin/go2tv" "$$@"\n' > $(APPDIR)/AppRun
	chmod +x $(APPDIR)/AppRun

	# Desktop entry and icon
	# Use provided desktop file and 512x512 icon
	cp $(DESKTOP_SRC) $(DESKTOP_APPDIR)
	cp $(ICON_SRC) $(ICON_APPDIR)
	# Also place a desktop file and icon at AppDir root as required by appimagetool
	cp $(DESKTOP_SRC) $(DESKTOP_ROOT)
	cp $(ICON_SRC) $(ICON_ROOT)

	# AppStream metadata (removes appimagetool warning)
	mkdir -p $(APPDATA_APPDIR)
	cp $(APPDATA_SRC) $(APPDATA_APPDIR)/

	# Ensure Exec and Icon fields are correct inside the desktop file
	sed -i 's/^Exec=.*/Exec=go2tv/g; s/^Icon=.*/Icon=go2tv/g' $(DESKTOP_APPDIR)
	sed -i 's/^Exec=.*/Exec=go2tv/g; s/^Icon=.*/Icon=go2tv/g' $(DESKTOP_ROOT)

	# Fetch appimagetool if missing
	set -e; \
	if [ ! -x $(APPIMAGETOOL) ]; then \
		URL="https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-$(ARCH).AppImage"; \
		curl -fsSL "$$URL" -o $(APPIMAGETOOL) || wget -q -O $(APPIMAGETOOL) "$$URL"; \
		chmod +x $(APPIMAGETOOL); \
	fi; \
	if [ ! -x $(APPIMAGETOOL) ]; then echo "appimagetool missing: $(APPIMAGETOOL)"; exit 1; fi; \
	if [ "$$(wc -c < $(APPIMAGETOOL))" -lt 1000000 ]; then echo "appimagetool download invalid: $(APPIMAGETOOL)"; exit 1; fi

	# Build the AppImage
	( cd $(BUILD_DIR) && ./appimagetool AppDir "$(notdir $(APPIMAGE_OUT))" ) && echo "AppImage created at $(APPIMAGE_OUT)"

	# Clean up ffmpeg build/download files
	rm -rf $(FFMPEG_STATIC_DIR) $(FFMPEG_STATIC_ARCHIVE) $(BUILD_DIR)/ffmpeg-src $(BUILD_DIR)/ffmpeg.tar.xz
