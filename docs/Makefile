.PHONY: all deploy clean

all: public public/index.html

public:
	git worktree add public gh-pages

public/index.html: public
	hugo

deploy:
	cd public && \
	git add --all && \
	git commit -m "Deploy to gh-pages" && \
	git push origin gh-pages

# Removing the actual public directory confuses git and will require a git
# worktree prune to fix
clean:
	rm -rf public/*
