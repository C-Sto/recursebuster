# RecurseBuster

It's like gobuster, but recursive!

I wanted a recursive directory brute forcer that was fast, and had certain features. It didn't exist, so I started writing this. In reality, I'll probably merge a lot of the functionality into github.com/swarley7/gograbber since that solves a similar problem and has cool features that I don't want to implement (phantomjs, ugh). For now, here it be!

## Installation

ye olde go get install should work. Same command to update:

```
go get -u github.com/c-sto/recursebuster
```

## Usage

I wanted it to be fairly straight forward, but scope creep etc. Basic usage is just like gobuster:

```
recursebuster -u https://google.com -w wordlist.txt
```

This will run a recursive-HEAD-spider-assissted search with a single thread on google.com using the wordlist specified above. Results will print to screen, but more importantly, will be written to a file 'busted.txt'.

## Features

### HEAD Based Checks

For servers the support it, HEAD based checks speed up content discovery considerably, since no body is required to be transferred. The default logic is to use a HEAD reqeust to determine if something exists. If it seems to exist, a GET is sent to retreive and verify.

### Recursion

When a directory is identified, it gets added to the queue to be brute-forced. By default, one directory is brute-forced at a time, mostly for sanity checks, but this can be increased to as many as you like.

### Spider Assistance

Since we are getting the page content anyway, why not use it to our advantage? Some basic checks are done to look for links on the page, these links are added, and any directories identified added too.

### Speed

Gobuster is pretty fast when you smash -t 200, but who would do that? One of my goals for this was to keep performance on-par with gobuster where possible. On most webservers, recursebuster seems to be faster, even though it sends both a HEAD and a GET request. This means you will hit WAF limits really quickly, and is why by default it's -t 1.
