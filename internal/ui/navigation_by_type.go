package ui

import (
	"net/http"
	"slices"
	"strings"
)

var menuObjectTypes = []struct {
	title string
	icon  string
}{
	{"Документы", "file-text"},
	{"Справочники", "book-open"},
	{"Отчёты", "chart-column"},
	{"Обработки", "settings"},
	{"Журналы", "notebook"},
	{"Регистры", "table"},
}

func menuObjectType(kind string) int {
	switch kind {
	case "document":
		return 0
	case "catalog", "accounts", "chartofaccounts":
		return 1
	case "report":
		return 2
	case "journal":
		return 4
	case "register", "inforeg", "accountreg":
		return 5
	default:
		return 3
	}
}

func menuNodeFromItem(item navItem) navTreeNode {
	return navTreeNode{ID: item.ID, DOMID: item.DOMID, Icon: item.Icon, Label: item.Label, URL: item.URL,
		TargetKind: item.TargetKind, TargetAction: item.TargetAction}
}

func menuSortKey(label string) string {
	return strings.ReplaceAll(strings.ToLower(label), "ё", "е")
}

func sortTypedMenu(nodes []navTreeNode, documents bool) {
	slices.SortStableFunc(nodes, func(a, b navTreeNode) int {
		if documents && (a.TargetAction == "create") != (b.TargetAction == "create") {
			if a.TargetAction == "create" {
				return -1
			}
			return 1
		}
		return strings.Compare(menuSortKey(a.Label), menuSortKey(b.Label))
	})
}

// navigationByType operates on already permission-filtered menu entries.
// Report folders are retained. Some 1C reports open through processors
// or pages and are explicitly placed in Reports or its KPI folder.
func (s *Server) navigationByType(r *http.Request, sub string, groups []navGroup) []navGroup {
	const reports = 2
	lang, context := s.resolveLang(r), navContext(sub)
	buckets := make([][]navTreeNode, len(menuObjectTypes))
	add := func(node navTreeNode, reportNodes *[]navTreeNode, inReports bool) {
		kind := menuObjectType(node.TargetKind)
		if kind == reports || (inReports && (node.TargetKind == "processor" || node.TargetKind == "page")) {
			*reportNodes = append(*reportNodes, node)
		} else {
			buckets[kind] = append(buckets[kind], node)
		}
	}
	var regroup func([]navTreeNode, bool) []navTreeNode
	regroup = func(nodes []navTreeNode, inReports bool) []navTreeNode {
		var out []navTreeNode
		for _, node := range nodes {
			if node.URL != "" {
				add(node, &out, inReports)
				continue
			}
			node.Nodes = regroup(node.Nodes, inReports || strings.EqualFold(node.Label, "KPI"))
			if len(node.Nodes) > 0 {
				sortTypedMenu(node.Nodes, false)
				out = append(out, node)
			}
		}
		return out
	}
	for _, group := range groups {
		title := menuSortKey(group.Kind)
		inReports := title == menuSortKey(menuObjectTypes[reports].title) ||
			title == menuSortKey(s.tr(lang, menuObjectTypes[reports].title))
		buckets[reports] = append(buckets[reports], regroup(group.Nodes, inReports)...)
		for _, item := range group.Items {
			add(menuNodeFromItem(item), &buckets[reports], inReports)
		}
		for _, folder := range group.Groups {
			node := navTreeNode{ID: folder.ID, DOMID: folder.DOMID, Icon: folder.Icon, Label: folder.Kind,
				CollapseKey: "navtree:" + navDOMID(currentUserLogin(r)+"\x00"+context, folder.ID)}
			for _, item := range folder.Items {
				add(menuNodeFromItem(item), &node.Nodes, inReports || strings.EqualFold(folder.Kind, "KPI"))
			}
			if len(node.Nodes) > 0 {
				sortTypedMenu(node.Nodes, false)
				buckets[reports] = append(buckets[reports], node)
			}
		}
	}
	var out []navGroup
	for i, info := range menuObjectTypes {
		nodes := buckets[i]
		if len(nodes) == 0 {
			continue
		}
		sortTypedMenu(nodes, i == 0)
		id := "cfg:menu-type:" + info.icon
		out = append(out, navGroup{ID: id, DOMID: navDOMID(context, id), Kind: s.tr(lang, info.title),
			Icon: info.icon, Gradient: true, Recursive: true, Open: true, Nodes: nodes,
			CollapseKey: "navtree:" + navDOMID(currentUserLogin(r)+"\x00"+context, id)})
	}
	return out
}
