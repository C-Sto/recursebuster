# RecurseBuster

It's like gobuster, but recursive!

I wanted a recursive directory brute forcer that was fast, and had certain features. It didn't exist, so I started writing this. In reality, I'll probably merge a lot of the functionality into github.com/swarley7/gograbber since that solves a similar problem and has cool features that I don't want to implement (PhantomJS, ugh). For now, here it be!

## Installation

Ye olde go get install should work. Same command to update:

```
go get -u github.com/c-sto/recursebuster
```

## Usage

I wanted it to be fairly straightforward, but scope creep etc. Basic usage is just like gobuster:

```
recursebuster -u https://google.com -w wordlist.txt
```

This will run a recursive-HEAD-spider-assisted search with a single thread on google.com using the wordlist specified above. Results will print to screen, but more importantly, will be written to a file 'busted.txt'.

## Features

### HEAD Based Checks

For servers the support it, HEAD based checks speed up content discovery considerably, since no body is required to be transferred. The default logic is to use a HEAD request to determine if something exists. If it seems to exist, a GET is sent to retrieve and verify.

### Recursion

When a directory is identified, it gets added to the queue to be brute-forced. By default, one directory is brute-forced at a time, mostly for sanity checks, but this can be increased to as many as you like.

### Spider Assistance

Since we are getting the page content anyway, why not use it to our advantage? Some basic checks are done to look for links on the page, these links are added, and any directories identified added too.

### Speed

Gobuster is pretty fast when you smash -t 200, but who would do that? One of my goals for this was to keep performance on-par with gobuster where possible. On most web servers, recursebuster seems to be faster, even though it sends both a HEAD and a GET request. This means you will hit WAF limits really quickly, and is why by default it's -t 1.

## Usage args

Idk why you might want these, just run it with -h. Here they are anyway:

```
  -all
        Show and write the result of all checks
  -bad string
        Responses to consider 'bad' or 'not found'. Comma-separated This works the opposite way of gobuster! (default "404")
  -blacklist string
        Blacklist of prefixes to not check. Will not check on exact matches.
  -canary string
        Custom value to use to check for wildcards
  -clean
        Output clean URLs to the output file for easy loading into other tools and whatnot.
  -cookies string
        Any cookies to include with requests. This is smashed into the cookies header, so copy straight from burp I guess.
  -debug
        Enable debugging
  -dirs int
        Maximum directories to perform busting on concurrently NOTE: directories will still be brute forced, this setting simply directs how many should be concurrently bruteforced (default 1)
  -ext string
        Extensions to append to checks. Multiple extensions can be specified, comma separate them.
  -k    Ignore SSL check
  -len
        Show and write the length of the response
  -noget
        Do not perform a GET request (only use HEAD request/response)
  -o string
        Local file to dump into (default "./busted.txt")
  -p string
        Proxy configuration options in the form ip:port eg: 127.0.0.1:9050
  -ratio float
        Similarity ratio to the 404 canary page. (default 0.95)
  -redirect
        Follow redirects
  -spider
        Search the page body for links, and directories to add to the spider queue.
  -t int
        Number of concurrent threads (default 1)
  -timeout int
        Timeout (seconds) for HTTP/TCP connections (default 20)
  -u string
        Url to spider
  -ua string
        User agent to use when sending requests. (default "RecurseBuster/1.0.0")
  -w string
        Wordlist to use for bruteforce. Blank for spider only
  -whitelist string
        Whitelist of domains to include in brute-force
```

Credits:

OJ/TheColonial: Hack the planet!!!!

Swarley: Hack the planet!!!!!

Hackers: Hack the planet!!!!
