# Contributing Guide

## A small thank you

Thanks for your interest in contributing to DCS Real Weather! This project
started a personal hobby project for my own group's DCS server, and it's now
grown to a small community of users. The feedback from everyone who has used
this tool has helped to improve it across the board, so your interest in
continuing that trend is greatly appreciated.

## Why this guide exists

This is a small project with a maintaining team of one (me), so reading these
guidelines will help me efficiently implement your feedback and increase the
chances of your feature request being added.

## Ways to contribute

There are many ways you can contribute to this project, and experience
programming is not required to be helpful! Ways of contributing include
improving the documentation, opening issues with bug reports or feature
requests, and writing code to be included in the next release of DCS Real
Weather.

### Documentation improvements

Documentation is a crucial aspect of any successful project, and unfortunately
I haven't spent much time creating a comprehensive guide on how to use DCS Real
Weather. If you would like to help make the tool easier for others to understand
and use, please feel empowered to do so. You can do this in multiple ways:

1. Add directly to the README.md and create a pull request. This may require
some understanding of GitHub and Git, but if you are familiar and would like
to get credit in the contributors section of GitHub, this is a great way to
do so!
2. Unsure how to create a pull request but stil want to suggest some
improvements to the README? That's OK too! You can always create a new issue
and suggest edits or additions there.
3. Go the extra mile and create a text or video guide and post it wherever!
This project could definitely benefit from someone willing to write a detailed
guide on how to set up DCS Real Weather and incorporate it into a server. If you
do this, please let me know by opening an issue and linking to your guide, and
if it's useful, I may include a link to it in the project README :-).

### Bug reports and feature requests

This is probably the easiest way to contribute to the project. If you are a user
of DCS Real Weather and think of a way to improve the tool, or if you encounter
a bug in your use, please let me know about it by creating an issue. For bug
reports, please be detailed! At minimum it is helpful if you can post the
following information:

- What did you do?
- What did you expect to happen?
- What actually happened?
- Please upload any relevant files to the issue. At minimum your mission file,
log file created by real weather, and config are most useful. Be sure to remove
any API keys from the config. Feel free to also include screenshots, video, etc
if you feel it would be helpful.

For feature requests, please layout your request with sufficient detail and
take the time to explain why you believe your feature would improve the tool.
Additionally, please understand that this is still a hobby project for me, and
if you request a large or hard-to-implement feature, it may not get done for a
long time, or at all. If this is the case, it may be better for you to take a
go at implementing the feature yourself, and opening a pull request ([see
contributing code](#contributing-code))

### Contributing code

Have some experience programming and want to contribute directly? Pull requests
are welcome. When creating your pull request, please still follow the
guidelines in [bug reports and feature
requests](#bug-reports-and-feature-requests) if you are opening a PR to fix a
bug or add a new feature. Additionally if your changes are large in scope, it
may be best to open an issue first to discuss what you plan to change prior to
doing the work.

Real Weather tries to follow common practices for Go development, so if you have
some experience with Go, it should be straightforward to jump in. All code
documentation is done in comments within each module. To build the project, see
the different options inside the Makefile. There is some code generation
necessary with `go generate ./...`, and the versioninfo module also contains a
helper program to generate versioning information. The version number can be
bumped with `go run versioninfo/generate/generate.go <version> <versioninfo
path>`, for example: `go run versioninfo/generate/generate.go v1.2.3
./versioninfo`.

Feel free to reach out in the [Discord](https://discord.com/invite/mjr2SpFuqq)
for additional help getting setup.

### Sponsoring

Financial contributions are never expected, so I do not offer any incentives for
donating. However, if you choose to do so, please know that I greatly appreciate
it. There are a couple options available via the sponsor buttons on the project
page. Donations through GitHub directly are preferred as BMAC does take a small
cut of your donation.

![sponsor options](/docs/img/sponsor_options.png)
