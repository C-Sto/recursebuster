# RecurseBuster_priv



Things Must do before pub release:

 - [] improve readme

Things OK Post-release:

- [] Deal with redirects more smarterer
- [] Have better debug output 
- [] Add verbosity levels to filter out messy output
- [] Ability to kill certain directory interactively
- [] Show current directories being bruteforced
- [] Do a 'smart404' check on the HEAD response based on content length. Probably will make this 'false' by default, but it might speed up checks considerably.
- [] Add ability to specify bad and/or good headers

Unsure if I want to add:

- [] OPTIONS based scans
- [] Ability to load in a list of URL's to start from
- [] Sort output so that it's grouped into directory trees
- [] Ability to save content locally (basically mirror the target)

Goals:
We want to know what might be hidden on the web server, and to get a good idea as to how it's mapped.
To do this, we spider it (the 'smart' way). But not all pages are linked - so we also want to directory brute-force.

We want to check for soft 404's (a 200 response, but 404 content). Intelligent 404 detection is a pain to get right.
We also want to do it quickly, or at least be able to manage the speed.

Work as a spider, but with brute-force/wordlist features

it's a directory if:

- we receive a redirect with the same url, but with a / at the end
- we look at the url with / at the end, and don't get a redirect
- directory indexing is enabled, and we can see the content of the folder
- we can see it in the url path on a link on the page

todo:
handle 429 'too many requests' response better

Output should be a reasonably logical directory/file tree. It should look like:
```
/
/index.html
/test.txt
/robots.txt
/folder1/test.html
/folder2/test.html
/folder2/folder3/cats.jpg
/folder4/asdf.txt
```

#Benchmarks

20/6/18

```PS C:\Repo\github.com\oj\gobuster> go run main.go -w ..\..\c-sto\recursebuster_priv\wordlist.txt -u http://159.203.178.9/ -t 20

Gobuster v1.4.1              OJ Reeves (@TheColonial)
=====================================================
=====================================================
[+] Mode         : dir
[+] Url/Domain   : http://159.203.178.9/
[+] Threads      : 20
[+] Wordlist     : ..\..\c-sto\recursebuster_priv\wordlist.txt
[+] Status codes : 301,302,307,200,204
=====================================================
/index.html (Status: 200)
Total Tested: 325 Estimated speed: 32/s
/. (Status: 200)
Total Tested: 663 Estimated speed: 33/s
Total Tested: 1001 Estimated speed: 33/s
Total Tested: 1337 Estimated speed: 33/s
[!] Keyboard interrupt detected, terminating.
=====================================================```

```PS C:\Repo\github.com\c-sto\recursebuster_priv> go run main.go -u http://159.203.178.9/ -w .\wordlist.txt -t 20
====================
GoRecurseBuster V0.0.4
Poorly hacked together by C_Sto
Heavy influence from Gograbber, thx Swarlz
====================
INFO: 2018/06/20 07:53:36 Starting...
INFO: 2018/06/20 07:53:37 Dirbusting http://159.203.178.9/
GOOD: 2018/06/20 07:53:37 Found http://159.203.178.9/ [200 OK]
GOOD: 2018/06/20 07:53:38 Found http://159.203.178.9/index.html [200 OK]
GOOD: 2018/06/20 07:53:39 Found http://159.203.178.9/. [200 OK]
GOOD: 2018/06/20 07:53:40 Found http://159.203.178.9/.htaccess [403 Forbidden]
GOOD: 2018/06/20 07:53:41 Found http://159.203.178.9/.htaccess/ [403 Forbidden]
INFO: 2018/06/20 07:53:41 Wildcard repsonse detected, skipping dirbusting of http://159.203.178.9/.htaccess/
GOOD: 2018/06/20 07:53:45 Found http://159.203.178.9/.html [403 Forbidden]
GOOD: 2018/06/20 07:53:46 Found http://159.203.178.9/.html/ [403 Forbidden]
INFO: 2018/06/20 07:53:46 Wildcard repsonse detected, skipping dirbusting of http://159.203.178.9/.html/
GOOD: 2018/06/20 07:53:49 Found http://159.203.178.9/.php [403 Forbidden]
GOOD: 2018/06/20 07:53:50 Found http://159.203.178.9/.php/ [403 Forbidden]
INFO: 2018/06/20 07:53:50 Wildcard repsonse detected, skipping dirbusting of http://159.203.178.9/.php/
exit status 27:53:56 Total Tested: 1255 Estimated speed: 68/s```