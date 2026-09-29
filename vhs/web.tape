# serra web - the tiny web UI on top of the same data
Output ../imgs/web.gif
Source common.tape
Set Height 200
Source setup.tape

Type "./serra web --port 8080 &"
Enter
Sleep 2s
Type "curl -so /dev/null -w 'HTTP %{http_code}\n' localhost:8080/?set=usg"
Enter
Sleep 3s
Type "kill %1"
Enter
Sleep 2s
