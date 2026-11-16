import pathlib
B=pathlib.Path(__file__).parent
s=(B/'native-b29-enum-20261116.py').read_text('utf-8')
old="W=B/'b29-native-enum-attempt1-20261116'";assert s.count(old)==1
s=s.replace(old,"W=B/'b29-native-enum-attempt2-20261116'")
anchor=" cli(['user','add',LOGIN,'--password-stdin'"
assert s.count(anchor)==1
s=s.replace(anchor," cli(['user','add','own_b29_bootstrap','--admin','--password-stdin','--project',str(P),'--sqlite',str(DB)],(secrets.token_urlsafe(32)+'\\n').encode())\n"+anchor)
p=B/'native-b29-enum-attempt2-20261116.py';assert not p.exists();p.write_text(s,encoding='utf-8')
print('new helper: bootstrap admin first; browser still logs in as nonadmin; SDK unchanged')
