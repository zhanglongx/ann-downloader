# ann-downloader

This is a Chinese stock market annual report downloader.

## Build

Follow the general steps of golang program compilation.

```bash
    $ go build .
```

## Usage

```bash
    $ ./ann-downloader [OPTIONS] <SYMBOL>
```

Generally, ann-downloader will download announcements from [cninfo](https://www.cninfo.com.cn):

- A target directory will be mkdir-ed under the default dir prefix. All announcements will be downloaded into the target directory.

- \<SYMBOL\> supports stock, name, pinyin. If there are duplicates, all items will be downloaded.

- Only supports download year announcements now. Announcements of the last two years will be downloaded. Here is the definition of the last two years: if the month of the current time is April and before, then the last two years are counted from the 3 years ago, otherwise from the previous year.

See \`./ann-downloader --help\` for more.

## Known Issues

- [cninfo](https://www.cninfo.com.cn) has some anti-crawler mechanism, so a common error is http connection error. This is usually solved by running it again after an interval of 1 minute.
