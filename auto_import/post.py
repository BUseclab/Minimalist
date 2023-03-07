import requests
from requests_toolbelt import MultipartEncoder

def prRed(skk): print("\033[91m {}\033[00m" .format(skk))
def prGreen(skk): print("\033[92m {}\033[00m" .format(skk))


url = "http://localhost:8086/admin/import_coverage/import"

mp = MultipartEncoder(fields = {'fk_version_id':'1', 'original_path_prefix':'', 'new_path_prefix':'/var/www/html/4.0.0/', 'test_name':'artifact', 'file_type':'function_coverage','file':('allowed.txt', open("./allowed.txt","rb"))})

prRed("Uploading Minimalist result ---- takes up to 20 minutes")
r = requests.post(url = url, data = mp, headers={'Content-Type':mp.content_type})

prGreen("visit http://localhost:8086/admin/ to debloat phpMyAdmin")
