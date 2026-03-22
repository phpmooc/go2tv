//go:build !(android || ios)

package gui

import (
	"container/ring"
	"context"
	"embed"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	fynetooltip "github.com/dweymouth/fyne-tooltip"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
	"go2tv.app/go2tv/v2/castprotocol"
	"go2tv.app/go2tv/v2/devices"
	"go2tv.app/go2tv/v2/httphandlers"
	"go2tv.app/go2tv/v2/rtmp"
	"go2tv.app/go2tv/v2/soapcalls"
	"go2tv.app/go2tv/v2/utils"
	"go2tv.app/screencast/hls"
)

// FyneScreen .
type FyneScreen struct {
	tempFiles                []string
	SelectInternalSubs       *widget.Select
	CurrentPos               binding.String
	EndPos                   binding.String
	serverStopCTX            context.Context
	cancelServerStop         context.CancelFunc
	Current                  fyne.Window
	cancelEnablePlay         context.CancelFunc
	PlayPause                *widget.Button
	Debug                    *debugWriter
	VolumeUp                 *widget.Button
	SkipPreviousButton       *widget.Button
	SkipNextButton           *widget.Button
	tvdata                   *soapcalls.TVPayload
	tabs                     *container.AppTabs
	CheckVersion             *widget.Button
	SubsText                 *widget.Entry
	CustomSubsCheck          *widget.Check
	NextMediaCheck           *widget.Check
	LoopSelectedCheck        *widget.Check
	TranscodeCheckBox        *widget.Check
	ScreencastCheckBox       *widget.Check
	Stop                     *widget.Button
	DeviceList               *deviceList
	httpserver               *httphandlers.HTTPserver
	MediaText                *widget.Entry
	ExternalMediaURL         *widget.Check
	SkinNextOnlySameTypes    bool
	GaplessMediaWatcher      func(context.Context, *FyneScreen, *soapcalls.TVPayload)
	SlideBar                 *tappedSlider
	MuteUnmute               *widget.Button
	VolumeDown               *widget.Button
	selectedDevice           devType
	selectedDeviceType       string
	chromecastClient         *castprotocol.CastClient // Active Chromecast connection
	chromecastActionID       uint64
	imageAutoSkipID          uint64
	State                    string
	mediafile                string
	version                  string
	eventURL                 string
	subsfile                 string
	controlURL               string
	renderingControlURL      string
	connectionManagerURL     string
	currentmfolder           string
	ffmpegPath               string
	ffmpegSeek               int
	castingMediaType         string  // MIME type of currently casting media (e.g., "image/jpeg", "video/mp4")
	mediaDuration            float64 // Actual media duration in seconds (from ffprobe, for transcoded streams)
	chromecastCheckedFile    string  // Tracks which file was already auto-checked for Chromecast compatibility
	systemTheme              fyne.ThemeVariant
	mediaFormats             []string
	audioFormats             []string
	videoFormats             []string
	imageFormats             []string
	muError                  sync.RWMutex
	mu                       sync.RWMutex
	ffmpegPathChanged        bool
	Medialoop                bool
	sliderActive             bool
	dlnaSeekRestart          bool
	imageAutoSkipTimeout     int
	Transcode                bool
	Screencast               bool
	screencastPrevTranscode  bool
	screencastPrevExternal   bool
	screencastPrevManualSubs bool
	screencastPrevLoop       bool
	screencastPrevNext       bool
	screencastPrevMediaText  string
	screencastPrevMediaFile  string
	screencastSession        *hls.Session
	screencastMu             sync.Mutex
	ErrorVisible             bool
	Hotkeys                  bool
	hotkeysSuspendCount      int32
	MediaBrowse              *widget.Button
	QueueButton              *widget.Button
	ClearMedia               *widget.Button
	SubsBrowse               *widget.Button
	SessionQueue             *SessionQueue
	queueWindow              fyne.Window
	queueList                *widget.List
	queueHeader              *widget.Label
	queueDetails             *widget.Label
	queuePlayNowButton       *widget.Button
	queueRemoveButton        *widget.Button
	queueMoveUpButton        *widget.Button
	queueMoveDownButton      *widget.Button
	queueClearButton         *widget.Button
	queueSelectedIndex       int
	lastQueueTapIndex        int
	lastQueueTapAt           time.Time
	ActiveDeviceLabel        *widget.Label
	ActiveDeviceCard         *widget.Card
	rtmpServer               *rtmp.Server
	rtmpServerCheck          *widget.Check
	transcodeToolTipCheck    *ttwidget.Check
	screencastToolTipCheck   *ttwidget.Check
	rtmpServerToolTipCheck   *ttwidget.Check
	rtmpURLCard              *widget.Card
	rtmpURLEntry             *widget.Entry
	rtmpKeyEntry             *widget.Entry
	rtmpHLSURL               string // The local HLS HLS URL
	rtmpPrevExternalMediaURL bool
	rtmpPrevLoop             bool
	rtmpPrevMediaText        string
	rtmpPrevMediaFile        string
	imageAutoSkipMediaPath   string
	imageAutoSkipCancel      context.CancelFunc
	rtmpMu                   sync.Mutex
	resumeSession            resumePlaybackSession
}

type debugWriter struct {
	ring *ring.Ring
}

type devType struct {
	name        string
	addr        string
	deviceType  string
	isAudioOnly bool
}

func (s *FyneScreen) updateFFmpegDependentCheckTooltips() {
	if s == nil {
		return
	}

	ffmpegMissing := utils.CheckFFmpeg(s.ffmpegPath) != nil
	toolTipMsg := lang.L("ffmpeg is required. install it or update ffmpeg path in Settings")

	setToolTip := func(ttCheck *ttwidget.Check, baseCheck *widget.Check) {
		if ttCheck == nil || baseCheck == nil {
			return
		}
		if ffmpegMissing && baseCheck.Disabled() {
			ttCheck.SetToolTip(toolTipMsg)
			return
		}
		ttCheck.SetToolTip("")
	}

	setToolTip(s.transcodeToolTipCheck, s.TranscodeCheckBox)
	setToolTip(s.screencastToolTipCheck, s.ScreencastCheckBox)
	setToolTip(s.rtmpServerToolTipCheck, s.rtmpServerCheck)
}

func (f *debugWriter) Write(b []byte) (int, error) {
	f.ring.Value = string(b)
	f.ring = f.ring.Next()
	return len(b), nil
}

//go:embed translations
var translations embed.FS

// Start .
func Start(ctx context.Context, s *FyneScreen) {
	if s == nil {
		return
	}

	if s.tempFiles == nil {
		s.tempFiles = make([]string, 0)
	}

	defer func() {
		for _, file := range s.tempFiles {
			os.Remove(file)
		}
	}()

	w := s.Current
	w.SetOnDropped(onDropFiles(s))

	tabs := container.NewAppTabs(
		container.NewTabItem("Go2TV", container.NewPadded(mainWindow(s))),
		container.NewTabItem(lang.L("Settings"), container.NewPadded(settingsWindow(s))),
		container.NewTabItem(lang.L("About"), aboutWindow(s)),
	)

	s.Hotkeys = true
	tabs.OnSelected = func(t *container.TabItem) {
		if t.Text == "Go2TV" {
			s.Hotkeys = true
			if s.rtmpServer == nil && !s.Screencast {
				s.TranscodeCheckBox.Enable()
				if s.ScreencastCheckBox != nil && !s.Screencast {
					s.ScreencastCheckBox.Enable()
				}
				s.SlideBar.Enable()
			} else if s.Screencast {
				s.TranscodeCheckBox.Disable()
			}

			ffmpegErr := utils.CheckFFmpeg(s.ffmpegPath)
			if ffmpegErr != nil {
				s.TranscodeCheckBox.SetChecked(false)
				s.TranscodeCheckBox.Disable()
				if s.ScreencastCheckBox != nil {
					s.ScreencastCheckBox.SetChecked(false)
					s.ScreencastCheckBox.Disable()
				}
				setInternalSubsDropdownNoSubs(s)
			}

			if ffmpegErr != nil {
				s.rtmpServerCheck.SetChecked(false)
				s.rtmpServerCheck.Disable()
			} else {
				if s.rtmpServer == nil {
					s.rtmpServerCheck.Enable()
				}
			}
			s.updateFFmpegDependentCheckTooltips()

			if s.ffmpegPathChanged {
				furi, err := storage.ParseURI("file://" + s.mediafile)
				if err == nil {
					selectMediaFile(s, furi)
				}
				s.ffmpegPathChanged = false
			}

			return
		}
		s.Hotkeys = false
	}

	s.ffmpegPathChanged = false

	if err := utils.CheckFFmpeg(s.ffmpegPath); err != nil {
		s.TranscodeCheckBox.Disable()
		if s.ScreencastCheckBox != nil {
			s.ScreencastCheckBox.Disable()
		}
		s.rtmpServerCheck.Disable()
	}
	s.updateFFmpegDependentCheckTooltips()

	s.tabs = tabs

	w.SetContent(fynetooltip.AddWindowToolTipLayer(tabs, w.Canvas()))
	minSize := tabs.MinSize()
	w.Resize(fyne.NewSize(fyne.Max(1000, minSize.Width), minSize.Height+(theme.Padding()*4)))
	w.CenterOnScreen()
	w.SetMaster()

	// Start Chromecast discovery in background
	go devices.StartChromecastDiscoveryLoop(ctx)

	go func() {
		<-ctx.Done()
		s.rtmpMu.Lock()
		if s.rtmpServer != nil {
			s.rtmpServer.Stop()
		}
		s.rtmpMu.Unlock()
		stopScreencastSession(s)
		os.Exit(0)
	}()

	w.SetOnClosed(func() {
		s.rtmpMu.Lock()
		if s.rtmpServer != nil {
			s.rtmpServer.Stop()
		}
		s.rtmpMu.Unlock()
		stopScreencastSession(s)
	})

	go silentCheckVersion(s)

	w.ShowAndRun()

}

// EmitMsg Method to implement the screen interface
func (p *FyneScreen) EmitMsg(a string) {
	fyne.Do(func() {
		switch a {
		case "Playing":
			setPlayPauseView("Pause", p)
			p.updateScreenState("Playing")
		case "Paused":
			setPlayPauseView("Play", p)
			p.updateScreenState("Paused")
		case "Stopped":
			stopAction(p)
		default:
			dialog.ShowInformation("?", "Unknown callback value", p.Current)
		}
	})
}

// SetMediaType Method to implement the screen interface
func (p *FyneScreen) SetMediaType(mediaType string) {
	p.mu.Lock()
	p.castingMediaType = mediaType
	p.mu.Unlock()
}

// Fini Method to implement the screen interface.
// Will only be executed when we receive a callback message,
// not when we explicitly click the Stop button.
func (p *FyneScreen) Fini() {
	fyne.Do(func() {
		if p.Screencast {
			return
		}

		gaplessOption := fyne.CurrentApp().Preferences().StringWithFallback("Gapless", "Disabled")

		// For Chromecast, ignore gapless setting (it's DLNA-specific)
		isChromecast := p.selectedDeviceType == devices.DeviceTypeChromecast

		// For Chromecast, reset state to Stopped so playAction doesn't interpret as pause
		if isChromecast {
			p.updateScreenState("Stopped")
		}

		if p.NextMediaCheck.Checked && (isChromecast || gaplessOption == "Disabled") {
			_, nextMediaPath, err := getNextMediaOrError(p)
			if err != nil {
				if isTraversalBoundaryError(err) {
					startAfreshPlayButton(p)
					return
				}
				check(p, err)
				startAfreshPlayButton(p)
				return
			}

			if err := setCurrentMediaPath(p, nextMediaPath); err != nil {
				check(p, err)
				startAfreshPlayButton(p)
				return
			}

			go playAction(p)
			return
		}
		// Main media loop logic
		if p.Medialoop {
			go playAction(p)
		}
	})
}

func check(s *FyneScreen, err error) {
	s.muError.Lock()
	defer s.muError.Unlock()

	fyne.Do(func() {
		if err != nil && !s.ErrorVisible {
			s.ErrorVisible = true
			cleanErr := strings.ReplaceAll(err.Error(), ": ", "\n")
			e := dialog.NewError(errors.New(cleanErr), s.Current)
			e.Show()
			e.SetOnClosed(func() {
				s.ErrorVisible = false
			})
		}
	})
}

var (
	errNoNextQueueMedia      = errors.New("no next queued media")
	errNoPreviousQueueMedia  = errors.New("no previous queued media")
	errNoPreviousFolderMedia = errors.New("no previous media file found in the current folder")
)

func getAdjacentMedia(screen *FyneScreen, delta int) (string, string, error) {
	if screen.hasSessionQueue() {
		return getAdjacentQueuedMedia(screen, delta)
	}

	return getAdjacentFolderMedia(screen, delta)
}

func getAdjacentQueuedMedia(screen *FyneScreen, delta int) (string, string, error) {
	queue, _ := screen.queueSnapshot()
	if queue == nil || len(queue.Items) == 0 {
		return "", "", errors.New(lang.L("queue is empty"))
	}

	if queue.CurrentIndex < 0 || queue.CurrentIndex >= len(queue.Items) {
		currentIndex := queue.indexByPath(screen.mediafile)
		if currentIndex == -1 {
			return "", "", errors.New(lang.L("current media file is not in the queue"))
		}
		queue.CurrentIndex = currentIndex
	}

	nextIndex := queue.adjacentIndex(delta, screen.SkinNextOnlySameTypes)
	if nextIndex == -1 {
		if delta < 0 {
			return "", "", errNoPreviousQueueMedia
		}

		return "", "", errNoNextQueueMedia
	}

	item := queue.Items[nextIndex]
	return item.BaseName, item.Path, nil
}

func getAdjacentFolderMedia(screen *FyneScreen, delta int) (string, string, error) {
	filedir := filepath.Dir(screen.mediafile)
	filelist, err := os.ReadDir(filedir)
	if err != nil {
		return "", "", err
	}

	files := make([]string, 0, len(filelist))
	currentType := screen.mediaKindForPath(screen.mediafile)

	for _, file := range filelist {
		fullPath := filepath.Join(filedir, file.Name())
		if !slices.Contains(screen.mediaFormats, filepath.Ext(fullPath)) {
			continue
		}

		if screen.SkinNextOnlySameTypes && currentType != screen.mediaKindForPath(fullPath) {
			continue
		}

		files = append(files, file.Name())
	}

	if len(files) == 0 {
		if delta < 0 {
			return "", "", errNoPreviousFolderMedia
		}

		return "", "", errors.New(lang.L("no next media file found in the current folder"))
	}

	currentIndex := slices.Index(files, filepath.Base(screen.mediafile))
	if currentIndex == -1 {
		return "", "", errors.New(lang.L("current media file not found in the current folder"))
	}

	switch {
	case delta < 0:
		if currentIndex == 0 {
			return "", "", errNoPreviousFolderMedia
		}

		prevName := files[currentIndex-1]
		return prevName, filepath.Join(filedir, prevName), nil
	case currentIndex+1 == len(files):
		nextName := files[0]
		return nextName, filepath.Join(filedir, nextName), nil
	default:
		nextName := files[currentIndex+1]
		return nextName, filepath.Join(filedir, nextName), nil
	}
}

func isTraversalBoundaryError(err error) bool {
	return errors.Is(err, errNoNextQueueMedia) ||
		errors.Is(err, errNoPreviousQueueMedia) ||
		errors.Is(err, errNoPreviousFolderMedia)
}

func getNextMediaOrError(screen *FyneScreen) (string, string, error) {
	return getAdjacentMedia(screen, 1)
}

func getPreviousMediaOrError(screen *FyneScreen) (string, string, error) {
	return getAdjacentMedia(screen, -1)
}

func autoSelectNextSubs(v string, screen *FyneScreen) {
	name, path := getNextPossibleSubs(v)
	screen.SubsText.Text = name
	screen.subsfile = path
	fyne.Do(func() {
		screen.SubsText.Refresh()
	})
}

func getNextPossibleSubs(v string) (string, string) {
	var name, path string

	possibleSub := v[0:len(v)-
		len(filepath.Ext(v))] + ".srt"

	if _, err := os.Stat(possibleSub); err == nil {
		name = filepath.Base(possibleSub)
		path = possibleSub
	}

	return name, path
}

func setPlayPauseView(s string, screen *FyneScreen) {
	if screen.cancelEnablePlay != nil {
		screen.cancelEnablePlay()
	}

	// Delay the update to avoid conflict with button tap animation.
	// Fyne's button tap animation doesn't synchronize with Refresh() calls,
	// causing visual artifacts. Delay by 300ms to let animation complete.
	go func() {
		fyne.Do(func() {
			// Check if we are casting an image
			isImage := false
			screen.mu.RLock()
			if strings.HasPrefix(screen.castingMediaType, "image/") {
				isImage = true
			}
			screen.mu.RUnlock()

			if isImage {
				screen.PlayPause.Disable()
				screen.PlayPause.SetIcon(theme.FileImageIcon())
				screen.PlayPause.SetText(lang.L("Image Casting") + "  ")
			} else {
				state := screen.getScreenState()
				if state == "Playing" || state == "Paused" {
					screen.PlayPause.Enable()
					switch s {
					case "Play":
						screen.PlayPause.SetText(lang.L("Play") + "  ")
						screen.PlayPause.SetIcon(theme.MediaPlayIcon())
					case "Pause":
						screen.PlayPause.SetText(lang.L("Pause") + "  ")
						screen.PlayPause.SetIcon(theme.MediaPauseIcon())
					}
				} else {
					// Stopped or initial state
					screen.PlayPause.Enable()

					if screen.rtmpServerCheck != nil && screen.rtmpServerCheck.Checked && screen.selectedDeviceType == devices.DeviceTypeChromecast {
						screen.PlayPause.SetText(lang.L("Start RTMP Session") + "  ")
					} else {
						screen.PlayPause.SetText(lang.L("Cast") + "  ")
					}
					screen.PlayPause.SetIcon(theme.MediaPlayIcon())
				}
			}
			screen.PlayPause.Refresh()
			screen.refreshTraversalControls()
		})
	}()
}

func setMuteUnmuteView(s string, screen *FyneScreen) {
	fyne.Do(func() {
		switch s {
		case "Mute":
			screen.MuteUnmute.Icon = theme.VolumeUpIcon()
		case "Unmute":
			screen.MuteUnmute.Icon = theme.VolumeMuteIcon()
		}
		screen.MuteUnmute.Refresh()
	})
}

// updateScreenState updates the screen state based on
// the emitted messages. The State variable is used across
// the GUI interface to control certain flows.
func (p *FyneScreen) updateScreenState(a string) {
	p.mu.Lock()
	p.State = a
	p.mu.Unlock()

	fyne.Do(func() {
		if p.DeviceList != nil {
			p.DeviceList.Refresh()
		}
		p.updateActiveDeviceView()
	})
}

func (p *FyneScreen) updateActiveDeviceView() {
	if p.ActiveDeviceCard == nil || p.ActiveDeviceLabel == nil {
		return
	}

	state := p.getScreenState()
	isActivePlayback := state == "Playing" || state == "Paused"

	if !isActivePlayback {
		p.ActiveDeviceCard.Hide()
		return
	}

	if p.selectedDevice.name != "" {
		p.ActiveDeviceLabel.SetText(p.selectedDevice.name)
		p.ActiveDeviceCard.Show()
	} else {
		p.ActiveDeviceCard.Hide()
	}
}

// getScreenState returns the current screen state
func (p *FyneScreen) getScreenState() string {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.State
}

func (p *FyneScreen) nextChromecastActionID() uint64 {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.chromecastActionID++
	return p.chromecastActionID
}

func (p *FyneScreen) isChromecastActionCurrent(actionID uint64) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.chromecastActionID == actionID
}

func (p *FyneScreen) currentChromecastActionID() uint64 {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.chromecastActionID
}

// checkChromecastCompatibility checks if loaded media needs transcoding for Chromecast.
// Auto-enables transcode checkbox if media is incompatible and FFmpeg is available.
// Only auto-enables once per file - tracks checked file to respect user's manual disable.
func (p *FyneScreen) checkChromecastCompatibility() {
	if p.selectedDeviceType != devices.DeviceTypeChromecast {
		return
	}
	if p.mediafile == "" {
		return
	}
	// Skip if we've already auto-checked this file (prevents re-enabling after user disables)
	if p.chromecastCheckedFile == p.mediafile {
		return
	}
	if err := utils.CheckFFmpeg(p.ffmpegPath); err != nil {
		return // Can't transcode anyway
	}

	// Only auto-enable transcoding for video files
	// Images and audio are natively supported by Chromecast
	ext := strings.ToLower(filepath.Ext(p.mediafile))
	if !slices.Contains(p.videoFormats, ext) {
		return // Not a video file, no need to check compatibility
	}

	info, err := utils.GetMediaCodecInfo(p.ffmpegPath, p.mediafile)
	if err != nil {
		return // Can't determine, let user decide
	}

	// Mark this file as checked (even if compatible) to avoid rechecking
	p.chromecastCheckedFile = p.mediafile

	if !utils.IsChromecastCompatible(info) {
		fyne.Do(func() {
			p.TranscodeCheckBox.SetChecked(true)
		})
		p.Transcode = true
	}
}

// NewFyneScreen creates and initializes a new FyneScreen instance with the provided version string.
func NewFyneScreen(version string) *FyneScreen {
	go2tv := app.NewWithID("app.go2tv.go2tv")

	// Hack. Ongoing discussion in https://github.com/fyne-io/fyne/issues/5333
	var content []byte
	switch go2tv.Preferences().String("Language") {
	case "中文(简体)":
		content, _ = translations.ReadFile("translations/zh.json")
	case "English":
		content, _ = translations.ReadFile("translations/en.json")
	}

	if content != nil {
		name := lang.SystemLocale().LanguageString()
		_ = lang.AddTranslations(fyne.NewStaticResource(name+".json", content))
	} else {
		_ = lang.AddTranslationsFS(translations, "translations")
	}

	go2tv.SetIcon(fyne.NewStaticResource("icon", go2TVIcon512))

	w := go2tv.NewWindow("Go2TV")
	currentDir, err := os.Getwd()
	if err != nil {
		currentDir = ""
	}

	dw := &debugWriter{
		ring: ring.New(1000),
	}

	ffmpegPath := func() string {
		if go2tv.Preferences().String("ffmpeg") != "" {
			path, err := utils.ResolveFFmpegPath(go2tv.Preferences().String("ffmpeg"))
			if err == nil {
				return path
			}
		}

		path, _ := utils.ResolveFFmpegPath("")
		return path
	}()

	return &FyneScreen{
		Current:            w,
		currentmfolder:     currentDir,
		ffmpegPath:         ffmpegPath,
		mediaFormats:       []string{".mp4", ".avi", ".mkv", ".mpeg", ".mov", ".webm", ".m4v", ".mpv", ".dv", ".mp3", ".flac", ".wav", ".m4a", ".jpg", ".jpeg", ".png"},
		imageFormats:       []string{".jpg", ".jpeg", ".png"},
		videoFormats:       []string{".mp4", ".avi", ".mkv", ".mpeg", ".mov", ".webm", ".m4v", ".mpv", ".dv"},
		audioFormats:       []string{".mp3", ".flac", ".wav", ".m4a"},
		version:            version,
		Debug:              dw,
		queueSelectedIndex: -1,
		lastQueueTapIndex:  -1,
	}
}

func onDropFiles(screen *FyneScreen) func(p fyne.Position, u []fyne.URI) {
	return func(p fyne.Position, u []fyne.URI) {
		if screen.Screencast {
			return
		}

		var mfiles, sfiles []fyne.URI

	out:
		for _, f := range u {
			if strings.HasSuffix(strings.ToUpper(f.Name()), ".SRT") {
				sfiles = append(sfiles, f)
				continue
			}

			for _, s := range screen.mediaFormats {
				if strings.HasSuffix(strings.ToUpper(f.Name()), strings.ToUpper(s)) {
					mfiles = append(mfiles, f)
					continue out
				}
			}
		}

		if len(sfiles) > 0 {
			screen.CustomSubsCheck.SetChecked(true)
			selectSubsFile(screen, sfiles[0])
		}

		if len(mfiles) > 0 {
			paths := make([]string, 0, len(mfiles))
			for _, mediaURI := range mfiles {
				paths = append(paths, mediaURI.Path())
			}
			check(screen, selectMediaPaths(screen, paths))
		}
	}
}
