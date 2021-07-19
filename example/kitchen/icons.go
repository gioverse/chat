package main

import (
	"gioui.org/widget"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

var NavBack *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.NavigationArrowBack)
	return icon
}()

var Send *widget.Icon = func() *widget.Icon {
	icon, _ := widget.NewIcon(icons.ContentSend)
	return icon
}()
