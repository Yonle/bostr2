#!/usr/bin/env bash

export CGO_ENABLED=1
export distsdir=dists/

init() {
	if ! (command -v go > /dev/null); then
		echo "go command not found."
		exit 1
	fi

	if ! (command -v tar > /dev/null); then
		echo "tar command not found."
		exit 1
	fi

	if ! (command -v zip > /dev/null); then
		echo "zip command not found."
		exit 1
	fi

	if [ -d $distsdir ]; then
		read -p "$distsdir directory exists. Remove to continue? [y/N]: " s
		case $s in
			y|Y)
				rm -rf $distsdir
				;;
			*)
				echo "Cannot crossbuild without existing $distsdir deleted."
				exit 1
				;;
		esac
	fi

	mkdir $distsdir
	cd $distsdir
}

preparedist() {
	distdir=bostr2-$1-$2
	mkdir $distdir
	cp ../config.example.yaml $distdir/
	cp ../LICENSE $distdir/
	cp ../README.md $distdir/

	cat > $distdir/INSTRUCTIONS.txt << EOL
Before running, Rename config.example.yaml to config.yaml,
Then edit the config with text editor before starting bouncer.

After editing, Start the bouncer by running:
  $ ./bostr2

Or launch bostr2.exe if you're on windows.

NOTE: If there's firewall, you must configure the firewall to allow
      incomming port to the bostr2 port.
EOL
}

maketarballs() {
	for distdir in bostr2-*; do
		if (echo $distdir | grep -q "windows"); then
			echo "--- Making zip ${distdir}.zip"
			zip -r ${distdir}.zip $distdir
		else
		        echo "--- Making tarball ${distdir}.tar.gz"
			tar -czf ${distdir}.tar.gz $distdir
		fi
	done
}

removedists() {
	echo "--- Cleaning up"
	for dist in bostr2-*.tar.gz; do
		rm -rf ${dist/.tar.gz}
	done

	for dist in bostr2-*.zip; do
		rm -rf ${dist/.zip}
	done
}

compile() {
	GOOS=$1
	GOARCH=$2

	distdir="bostr2-$1-$2"
	echo "--- Compiling for $GOOS/$GOARCH platform"
	local WINEX=""

	if [ "$GOOS" == "windows" ]; then
		WINEX=".exe"
	fi

	go build -o $distdir/bostr2${WINEX} ../;

	if [ "$?" != "0" ]; then
		echo "--- Compilation failed for $GOOS/$GOARCH. Destroying distdir..."
		rm -rf $distdir
	fi
}

close() {
	echo "--- Result:"
	ls
	echo "--- Tarballs & zips is available in $(pwd)"
}

crosscompile() {
	go tool dist list | sed 's/\// /g' | while read os arch; do
		preparedist $os $arch
		compile $os $arch
	done
}

init
crosscompile
maketarballs
removedists
close
