## chat

A framework for building chat interfaces with [gioui.org](https://gioui.org).

The canonical copy of this repository is hosted [on Sourcehut](https://git.sr.ht/~gioverse/chat). We also have:

- [An issue tracker.](https://todo.sr.ht/~gioverse/chat)
- [A mailing list.](https://lists.sr.ht/~gioverse/chat)

## Status

Work in progress. Expect breaking API changes often.

## Parts

This repo contains many packages with different goals:

- `list`: This package implements logic to allow a Gio list to scroll over arbitrarily large quantities of elements while keeping a fixed number in RAM.
- `layout`: This package exposes some layout types and functions helpful for building chat interfaces.
- `ninepatch`: A ninepatch image decoder.
- `widget`: This package exposes the state types for some helpful chat widgets.
- `widget/material`: This package exposes some material-design-themed components for building chat interfaces.
- `example/kitchen`: An example of using all of this parts of this module together to build a chat interface.
- `example/ninepatch`: An example of the ninepatch decoder.
- `example/unconfigured`: A demonstration of the default behavior of the `list.Manager` if no custom hooks are provided to it.

## Usage

See the `./example` directory for applications showcasing how to use the current
API.

In particular `./example/kitchen` tries to exercise the full range of this
module's features.

## License

Dual Unlicense/MIT, same as Gio
