lilliput
========

Jabong's Url Shortener

## HOW TO INSTALL GOLANG
<ol>
<li>Download the archive from <a href='http://golang.org/dl/'>here</a></li>
<li>Go to the folder containing downloaded file using terminal</li>
<li>Extract the file into /usr/local,creating a go tree in /usr/local/go by typing
<pre><code>tar -C /usr/local -xzf go$VERSION.$OS-$ARCH.tar.gz</code></pre>
    For example if you are installing Go version 1.2.1 for 64-bit x86 on Linux.
<pre><code>tar -C /usr/local -xzf go1.2.1.linux-amd64.tar.gz</code></pre></li>
<li>Add environment variables in .bashrc file</li>
 <ol><li>Go to $HOME direcitory by typing
<pre><code>cd</pre></code></li>
   <li>Open .bashrc file
   <pre><code>vi .bashrc</code></pre></li>
   <li>Place these lines in the end of the file(.bashrc)
<pre><code>export GOROOT=/usr/local/go
export GOPATH=$HOME/lilliput
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
export PATH=$PATH:$GOPATH/bin</code></pre></li>
   <li>Save and exit the file. Press "Esc" and then :wq</li>
   <li>Source the .bashrc file
<pre><code>source .bashrc </code></pre></li></ol></li>
<li>Test your installation by typing
<pre><code>go version</code></pre></li></ol>

## HOW TO INSTALL LILLIPUT
<li>Fork the repository by clicking the "Fork" button in the GitHub.com repository(apiary).</li>
<li>Clone your forked repo in $HOME
<pre><code>git clone https://github.com/jabong/lilliput</code></pre></li>
<li> Crete branches to build new features and test out ideas
<pre><code>git branch "branchname"
git checkout "branchname"</code></pre>
Alternatvely we can use<br>
<pre><code>git checkout -b "branchname"</code></pre></li>
<li>Deploy dependencies. Go to the directory $HOME/lilliput and type
<pre><code>make depends</code></pre></li>
<li>To build apiary executable file type
<pre><code>make build</code></pre></li></ol>
<li><b>Configuration:</b></li>
<li>Copy config.ini to dev.ini and change values as per your environment</li>
<pre>
	<code>
			[lilliput]
			// port number on which lilliput will be running
			port = 8989 
			// domain name which will be prepend in tinyurl
			domain = "http://127.0.0.1:8989/"
			[redis]
			server = "127.0.0.1"
			port = "6379"
			dbname = 0
	</code>
</pre>

# Sample Usage
<pre>
	<code>
		shell> curl --data "url=http://google.com" http://127.0.0.1:8989
	</code>
</pre>
###### Output
```json
	{"url":"http://127.0.0.1:8989/4P","err":false,"message":"","OrgUrl":"http://google.com"}
```
