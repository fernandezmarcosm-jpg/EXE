from pathlib import Path

p = Path("column_view_windows.go")
s = p.read_text(encoding="utf-8")
start = s.index("func columnViewVisibleColumns() []DatasetColumn {")
end = s.index("\n}\n\nfunc columnViewFilteredRecords", start) + 2
new = '''func columnViewVisibleColumns() []DatasetColumn {
\tif viewDataset == nil { return nil }
\tlimit := appSettings.MaxColumns
\tif limit < 1 { limit = 20 }
\tout := make([]DatasetColumn, 0, len(viewDataset.Columns))
\tfor _, c := range viewDataset.Columns {
\t\tif !c.Visible { continue }
\t\tif len(out) >= limit { break }
\t\tout = append(out, c)
\t}
\treturn out
}'''
s = s[:start] + new + s[end:]
old = '''\tif changed { _ = saveDatasetSettings(appSettings) }
\tcolumnViewBuildFilters()
\tcolumnViewRefresh()
\tuser32.NewProc("DestroyMenu").Call(m)
}'''
new2 = '''\tif changed { _ = saveDatasetSettings(appSettings) }
\tif changed {
\t\tuser32.NewProc("SendMessageW").Call(viewList, lvmSetItemCountEx, 0, 0)
\t\tcolumnViewDeleteColumns()
\t\tcolumnViewBuildFilters()
\t\tcolumnViewRefresh()
\t}
\tuser32.NewProc("DestroyMenu").Call(m)
}'''
if old not in s:
    raise SystemExit("show menu tail pattern not found")
s = s.replace(old, new2, 1)
p.write_text(s, encoding="utf-8")
