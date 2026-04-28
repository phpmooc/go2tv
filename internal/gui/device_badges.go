package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	ttwidget "github.com/dweymouth/fyne-tooltip/widget"
	"go2tv.app/go2tv/v2/devices"
	"go2tv.app/go2tv/v2/internal/devicecolors"
)

type badgePalette struct {
	fill   color.Color
	stroke color.Color
	text   color.Color
}

type deviceBadge struct {
	widget.BaseWidget
	background *canvas.Rectangle
	label      *canvas.Text
}

type deviceBadgeRenderer struct {
	badge   *deviceBadge
	objects []fyne.CanvasObject
}

type deviceRowRenderer struct {
	row     *deviceRow
	objects []fyne.CanvasObject
}

type deviceTitleLayout struct{}

type deviceRow struct {
	widget.BaseWidget
	leading  *widget.Icon
	trailing *widget.Icon
	name     *ttwidget.Label
	badges   *fyne.Container
	content  fyne.CanvasObject
}

func currentThemeVariant() fyne.ThemeVariant {
	app := fyne.CurrentApp()
	if app == nil {
		return theme.VariantDark
	}

	switch app.Preferences().StringWithFallback("Theme", "System Default") {
	case "Light":
		return theme.VariantLight
	case "Dark":
		return theme.VariantDark
	}

	return app.Settings().ThemeVariant()
}

func newDeviceBadge(text string, palette badgePalette) *deviceBadge {
	bg := canvas.NewRectangle(palette.fill)
	bg.CornerRadius = theme.InputRadiusSize()
	bg.StrokeColor = palette.stroke
	bg.StrokeWidth = 0

	label := canvas.NewText(text, palette.text)
	label.TextSize = theme.CaptionTextSize()
	label.TextStyle = fyne.TextStyle{Bold: true}
	label.Alignment = fyne.TextAlignCenter

	badge := &deviceBadge{
		background: bg,
		label:      label,
	}
	badge.ExtendBaseWidget(badge)
	return badge
}

func (b *deviceBadge) CreateRenderer() fyne.WidgetRenderer {
	return &deviceBadgeRenderer{
		badge:   b,
		objects: []fyne.CanvasObject{b.background, b.label},
	}
}

func (r *deviceBadgeRenderer) Destroy() {}

func (r *deviceBadgeRenderer) Layout(size fyne.Size) {
	r.badge.background.Resize(size)

	textSize := r.badge.label.MinSize()
	r.badge.label.Resize(textSize)
	r.badge.label.Move(fyne.NewPos(
		(size.Width-textSize.Width)/2,
		(size.Height-textSize.Height)/2,
	))
}

func (r *deviceBadgeRenderer) MinSize() fyne.Size {
	textSize := r.badge.label.MinSize()
	padX := theme.InnerPadding()
	padY := theme.InnerPadding() * 0.35
	return fyne.NewSize(textSize.Width+(padX*2), textSize.Height+(padY*2))
}

func (r *deviceBadgeRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *deviceBadgeRenderer) Refresh() {
	r.badge.background.CornerRadius = theme.InputRadiusSize()
	canvas.Refresh(r.badge.background)
	canvas.Refresh(r.badge.label)
}

func (deviceTitleLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) != 2 {
		return
	}

	name := objects[0]
	badges := objects[1]
	gap := theme.InnerPadding() * 0.75
	rightPad := theme.InnerPadding() * 0.5
	badgesMin := badges.MinSize()
	badgesWidth := badgesMin.Width
	availableWidth := size.Width - rightPad
	if availableWidth < 0 {
		availableWidth = 0
	}
	if badgesWidth > availableWidth {
		badgesWidth = availableWidth
	}

	nameWidth := availableWidth - badgesWidth
	if nameWidth > 0 {
		nameWidth -= gap
	} else {
		gap = 0
	}
	if nameWidth < 0 {
		nameWidth = 0
	}

	name.Resize(fyne.NewSize(nameWidth, size.Height))
	name.Move(fyne.NewPos(0, 0))

	badges.Resize(badgesMin)
	badges.Move(fyne.NewPos(nameWidth+gap, (size.Height-badgesMin.Height)/2))
}

func (deviceTitleLayout) MinSize(objects []fyne.CanvasObject) fyne.Size {
	if len(objects) != 2 {
		return fyne.Size{}
	}

	nameMin := objects[0].MinSize()
	badgesMin := objects[1].MinSize()
	gap := theme.InnerPadding() * 0.75
	rightPad := theme.InnerPadding() * 0.5

	return fyne.NewSize(nameMin.Width+gap+badgesMin.Width+rightPad, max(nameMin.Height, badgesMin.Height))
}

func (r *deviceRowRenderer) Destroy() {}

func (r *deviceRowRenderer) Layout(size fyne.Size) {
	r.row.content.Resize(size)
	r.row.updateNameToolTip()
}

func (r *deviceRowRenderer) MinSize() fyne.Size {
	return r.row.content.MinSize()
}

func (r *deviceRowRenderer) Objects() []fyne.CanvasObject {
	return r.objects
}

func (r *deviceRowRenderer) Refresh() {
	canvas.Refresh(r.row.content)
	r.row.updateNameToolTip()
}

func deviceBadgePalette(deviceType string) badgePalette {
	if currentThemeVariant() == theme.VariantLight {
		return badgePaletteFromDeviceColors(devicecolors.LightPalette(deviceType))
	}

	return badgePaletteFromDeviceColors(devicecolors.DarkPalette(deviceType))
}

func audioOnlyBadgePalette() badgePalette {
	if currentThemeVariant() == theme.VariantLight {
		return badgePaletteFromDeviceColors(devicecolors.AudioOnlyLightPalette())
	}

	return badgePaletteFromDeviceColors(devicecolors.AudioOnlyDarkPalette())
}

func badgePaletteFromDeviceColors(p devicecolors.Palette) badgePalette {
	return badgePalette{
		fill:   p.Fill,
		stroke: p.Stroke,
		text:   p.Text,
	}
}

func deviceBadgeObjects(item devType) []fyne.CanvasObject {
	badges := []fyne.CanvasObject{
		newDeviceBadge(lang.L(item.deviceType), deviceBadgePalette(item.deviceType)),
	}

	if item.isAudioOnly {
		badges = append(badges, newDeviceBadge(lang.L("Audio only"), audioOnlyBadgePalette()))
	}

	return badges
}

func newDeviceRow(leading, trailing fyne.Resource) *deviceRow {
	row := &deviceRow{
		name: ttwidget.NewLabel("Device Name"),
		badges: container.NewHBox(
			newDeviceBadge(lang.L(devices.DeviceTypeChromecast), deviceBadgePalette(devices.DeviceTypeChromecast)),
		),
	}
	row.name.Truncation = fyne.TextTruncateEllipsis

	if leading != nil {
		row.leading = widget.NewIcon(leading)
	}

	if trailing != nil {
		row.trailing = widget.NewIcon(trailing)
	}

	center := container.New(&deviceTitleLayout{}, row.name, row.badges)
	switch {
	case row.leading != nil && row.trailing != nil:
		row.content = container.NewBorder(nil, nil, row.leading, row.trailing, center)
	case row.leading != nil:
		row.content = container.NewBorder(nil, nil, row.leading, nil, center)
	case row.trailing != nil:
		row.content = container.NewBorder(nil, nil, nil, row.trailing, center)
	default:
		row.content = center
	}

	row.ExtendBaseWidget(row)
	return row
}

func (r *deviceRow) CreateRenderer() fyne.WidgetRenderer {
	return &deviceRowRenderer{
		row:     r,
		objects: []fyne.CanvasObject{r.content},
	}
}

func (r *deviceRow) setDevice(item devType) {
	r.name.SetText(item.name)
	r.badges.Objects = deviceBadgeObjects(item)
	r.badges.Refresh()
	r.updateNameToolTip()
}

func (r *deviceRow) setLeadingIcon(icon fyne.Resource) {
	if r.leading == nil {
		return
	}

	r.leading.SetResource(icon)
	r.leading.Refresh()
}

func (r *deviceRow) updateNameToolTip() {
	if r == nil || r.name == nil {
		return
	}

	usableWidth := r.name.Size().Width - (theme.InnerPadding() * 2)
	if usableWidth <= 0 {
		r.name.SetToolTip("")
		return
	}

	textWidth := fyne.MeasureText(
		r.name.Text,
		theme.SizeForWidget(theme.SizeNameText, r.name),
		r.name.TextStyle,
	).Width

	if textWidth > usableWidth {
		r.name.SetToolTip(r.name.Text)
		return
	}

	r.name.SetToolTip("")
}
