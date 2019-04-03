This package allows reading and writing [bettercap](https://bettercap.org)'s session recordings.

A recording archive is a gzip file containing reference Session and Events JSON objects and their changes stored as patches in order to keep the file size as small as possible. Loading a session file implies generating all the frames starting from the reference one by iteratively applying those "state patches" until all recorded frames are stored in memory. This is done to allow, UI side, to skip forward to a specific frame index without all intermediate states being computed at runtime.

## Example

See the `examples` folder.

## License

This package is made with ♥  by [evilsocket](https://github.com/evilsocket) and it's released under the GPL 3 license.
