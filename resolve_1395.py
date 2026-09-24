import io, re

pat = re.compile(r'<<<<<<< HEAD\n(.*?)=======\n(.*?)>>>>>>> origin/main\n', re.S)

# handlers_entity.go: main's response map (with selected_allowed) + PR's preview fields
p = 'internal/ui/handlers_entity.go'
s = io.open(p, encoding='utf-8').read()
def he(m):
    theirs = m.group(2)
    add = '''	response["preview"] = previewField
	response["previewHtml"] = previewHTML
'''
    # insert preview fields right before selected_allowed block in theirs
    anchor = '\tif choice != nil && choice.Selected != nil {'
    assert anchor in theirs
    return theirs.replace(anchor, add + anchor, 1)
s = pat.sub(he, s, count=1)
assert '<<<<<<<' not in s, p
io.open(p, 'w', encoding='utf-8', newline='').write(s)
print('handlers_entity.go combined')

# ui.js: keep both sides where blocks are additive; main wins on identical spots
p = 'internal/ui/static/ui.js'
s = io.open(p, encoding='utf-8').read()
def ui(m):
    ours, theirs = m.group(1), m.group(2)
    return ours + '\n' + theirs
n = 0
def ui2(m):
    global n
    n += 1
    return ui(m)
s = pat.sub(ui2, s)
print('ui.js hunks combined:', n)
assert '<<<<<<<' not in s
io.open(p, 'w', encoding='utf-8', newline='').write(s)

# docs: both sides
for p in ['docs/features.md', 'docs/forms.md']:
    s = io.open(p, encoding='utf-8').read()
    s = pat.sub(lambda m: m.group(1) + m.group(2), s)
    assert '<<<<<<<' not in s
    io.open(p, 'w', encoding='utf-8', newline='').write(s)
    print(p, 'both sides')
