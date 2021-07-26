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

### List

The `list` package API deserves some discussion. `list.Manager` handles the complex task of maintaining a sliding window of list elements atop an arbitrarily-long (maybe infinite) underlying list of content. This means that `list.Manager` handles all of the following:

- Requesting list content from its source of truth,
- Maintaining the proper ordering and deduplication of that content,
- Discarding the element data furthest away from the current list viewport when the list grows too large,
- Resolving dynamic attributes of list content,
- Injecting widgets between list elements (such as unread markers or date separators in chat),
- Allocating and persisting state for the heterogeneous list contents,
- And, laying out widgets for each element in the managed list.

`list.Manager` is able to accomplish all of the above in a generic way by reqiring a set of "hooks" provided by your application. These hooks supply application specific intelligence about the concrete types of your data, the way that your data relates to itself, and the way that your list elements should be presented to the user.

The required hooks are:

- `Loader`: a function that the `list.Manager` can invoke to load more elements from the source of truth for the list data. This function is expected to block during the load, and the parameters provided to it indicate the direction and relative position of the requested content within the source of truth.
- `Comparator`: a function used to sort list elements. It is provided two elements and returns whether the first sorts before the second.
- `Sythesizer`: a function that can transform the list elements from within the state management goroutine. Many applications may have list elements with dynamic properties. Those properties require and API call or database interaction to resolve, and you don't want to perform such blocking I/O from the layout goroutine. This hook provides a place to perform blocking queries and other data transformations without blocking the layout goroutine. In particular, this hook can return zero or more elements, meaning that it can choose to hide list elements or to insert other elements around them prior to layout.
- `Invalidator`: a function that can invalidate the window. This is necessary so that the `list.Manager` can ensure that Gio draws a new frame once it finishes a state update.
- `Allocator`: a function that accepts a `list.Element` and returns the appropriate state type for it. The returned state type will be persisted by the `list.Manager`. For instance, a `list.Element` that renders a button would return `*widget.Clickable` from this hook, so that it has somewhere to store its click state across frames.
- `Presenter`: a function that accepts a `list.Element`, some state from the `Allocator`, and returns a Gio `layout.Widget` that will lay out the list element as a widget.

As much work as possible is performed in a background state management goroutine so that the layout goroutine has no reason to block.

Here's a diagram showing how the various hooks work together:

![diagram](https://git.sr.ht/~gioverse/chat/blob/main/list/assets/dataflow-diagram.png)

For a relatively simple implementation of using all of the hooks together to build something useful, see `./example/carousel/`.

## License

Dual Unlicense/MIT, same as Gio
