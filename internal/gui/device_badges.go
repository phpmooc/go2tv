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
	bg.StrokeWidth = 1

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
	padX := theme.InnerPadding() * 1.5
	padY := theme.InnerPadding() * 0.6
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
		switch deviceType {
		case devices.DeviceTypeChromecast:
			return badgePalette{
				fill:   color.NRGBA{R: 0x00, G: 0xaa, B: 0x8d, A: 0xff},
				stroke: color.NRGBA{R: 0x00, G: 0x7d, B: 0x68, A: 0xff},
				text:   color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
			}
		case devices.DeviceTypeDLNA:
			return badgePalette{
				fill:   color.NRGBA{R: 0x6d, G: 0x97, B: 0xff, A: 0xff},
				stroke: color.NRGBA{R: 0x4a, G: 0x6f, B: 0xc9, A: 0xff},
				text:   color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
			}
		default:
			return badgePalette{
				fill:   color.NRGBA{R: 0x70, G: 0x70, B: 0x70, A: 0xff},
				stroke: color.NRGBA{R: 0x52, G: 0x52, B: 0x52, A: 0xff},
				text:   color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
			}
		}
	}

	switch deviceType {
	case devices.DeviceTypeChromecast:
		return badgePalette{
			fill:   color.NRGBA{R: 0x00, G: 0x84, B: 0x6f, A: 0x24},
			stroke: color.NRGBA{R: 0x00, G: 0xd2, B: 0xaf, A: 0xff},
			text:   color.NRGBA{R: 0x00, G: 0xe3, B: 0xbd, A: 0xff},
		}
	case devices.DeviceTypeDLNA:
		return badgePalette{
			fill:   color.NRGBA{R: 0x1c, G: 0x55, B: 0xb8, A: 0x1f},
			stroke: color.NRGBA{R: 0x6f, G: 0xad, B: 0xff, A: 0xff},
			text:   color.NRGBA{R: 0x8c, G: 0xbd, B: 0xff, A: 0xff},
		}
	default:
		return badgePalette{
			fill:   color.NRGBA{R: 0x55, G: 0x55, B: 0x55, A: 0x1f},
			stroke: color.NRGBA{R: 0x9c, G: 0x9c, B: 0x9c, A: 0xff},
			text:   color.NRGBA{R: 0xb8, G: 0xb8, B: 0xb8, A: 0xff},
		}
	}
}

func audioOnlyBadgePalette() badgePalette {
	if currentThemeVariant() == theme.VariantLight {
		return badgePalette{
			fill:   color.NRGBA{R: 0xd1, G: 0x83, B: 0x0d, A: 0xff},
			stroke: color.NRGBA{R: 0x9d, G: 0x5f, B: 0x05, A: 0xff},
			text:   color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: 0xff},
		}
	}

	return badgePalette{
		fill:   color.NRGBA{R: 0x9b, G: 0x62, B: 0x09, A: 0x1f},
		stroke: color.NRGBA{R: 0xff, G: 0xb1, B: 0x3b, A: 0xff},
		text:   color.NRGBA{R: 0xff, G: 0xc6, B: 0x69, A: 0xff},
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

	center := container.NewBorder(nil, nil, nil, row.badges, row.name)
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
