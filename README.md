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

- Only supports download year announcements now. If \`-year\` not specified, announcements of the last three years will be downloaded when exists. 

See './ann-downloader --help' for more.

## Known Issues

- [cninfo](https://www.cninfo.com.cn) has some anti-crawler mechanisms, so a common error is http connection error. This is usually solved by running it again after an interval of 1 minute.
