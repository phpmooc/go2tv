package gui

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
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

type deviceRow struct {
	widget.BaseWidget
	leading  *widget.Icon
	trailing *widget.Icon
	name     *widget.Label
	badges   *fyne.Container
	content  fyne.CanvasObject
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

func deviceBadgePalette(deviceType string) badgePalette {
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
		name: widget.NewLabel("Device Name"),
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

	center := container.NewVBox(row.name, row.badges)
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
	return widget.NewSimpleRenderer(r.content)
}

func (r *deviceRow) setDevice(item devType) {
	r.name.SetText(item.name)
	r.badges.Objects = deviceBadgeObjects(item)
	r.badges.Refresh()
}

func (r *deviceRow) setLeadingIcon(icon fyne.Resource) {
	if r.leading == nil {
		return
	}

	r.leading.SetResource(icon)
	r.leading.Refresh()
}
