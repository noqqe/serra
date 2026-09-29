# serra migrate - bring the database up to the expected schema
Output ../imgs/migrate.gif
Source common.tape
Set Height 200
Source setup.tape

Type "./serra migrate --status"
Enter
Sleep 3s
Type "./serra migrate"
Enter
Sleep 4s
