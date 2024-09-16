#!/bin/bash

install_azdoext() {
    # Determine OS and Architecture
    os=$(uname | tr '[:upper:]' '[:lower:]')
    arch=$(uname -m | tr '[:upper:]' '[:lower:]' | sed -e s/x86_64/amd64/)

    # Define download URL
    tar="azdoext-${os}-${arch}.tar.gz"
    url="https://github.com/rdalbuquerque/azdoext/releases/latest/download"

    # Defined filename
    version=$(curl -s https://api.github.com/repos/rdalbuquerque/azdoext/releases/latest | jq -r '.tag_name')
    filename="azdoext_${version}_$os-$arch"

    # azdoext user bin directory
    installdir=$HOME/.local/bin

    # Download and install
    echo "Downloading version ${version} of azdoext-$os-$arch..."
    curl -sL "$url/$tar" -o "/tmp/$tar"
    if [ ! -f "/tmp/$tar" ]; then
      echo "Error downloading azdoext-$os-$arch."
      return 1
    fi
    echo

    tar -xzf "/tmp/$tar" -C /tmp
    # Optionally remove the tar file after extracting
    # rm "/tmp/$tar"

    echo "Moving /tmp/$filename to $installdir/azdoext"
    mkdir -p $installdir && mv "/tmp/$filename" "$installdir/azdoext"

    # Add ~/.local/bin to PATH if it exists
    echo "Adding $installdir to PATH"
    if [ -d "$HOME/.local/bin" ] ; then
        PATH="$HOME/.local/bin:$PATH"
    fi

    # test installation
    azdoext --version
    if [ $? -eq 0 ]; then
      echo "azdoext installed successfully"
    else
      echo "Failed to install azdoext"
      return 1
    fi


    echo
}

# Execute the function
install_azdoext