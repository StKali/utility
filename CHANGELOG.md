## 20240423(v1.2.5)
- fix: calling the SetErrorPrefix function does not work

## 20231127(v1.2.4)
- fix: paths.SplitWithExt will use the leftmost dot as the delimiter when there are multiple dots in the file name.
- feat: returns 0 when the Min and Max parameters is empty

## 20231127(v1.2.3)
- fix: not compatible with go versions 1.18, 1.19, and 1.20

## 20231122(v1.2.2)
- feat: add rotating file/log support
- feat: add paths package
- fix: fix the panic caused by tool.RandInternalString when min and max are consistent

## 20231110(v1.2.1)
- feat: change the type of `LightCmd.Env` from `[]string` to `map[string]string`
- docs: add CHANGELOG.md file and update README.md
