
GO_SOURCE = Rake::FileList['**/*.go'].tap do |src|
  src.exclude %r{^vendor/}
  src.exclude do |f|
    `git ls-files '#{f}'`.empty?
  end
end

task default: :build

desc 'Clean things up'
task :clean do
  sh 'git clean -xfd'
end

desc 'Fetch all dependencies'
task :setup do
  sh 'go version'
  sh(
    *%w(
      go get -u
      github.com/Masterminds/glide
      github.com/mitchellh/gox
    )
  )
  sh 'glide install'
end

desc 'Lint the code'
task lint: %w(lint:vet lint:golint)

namespace :lint do
  task :vet do
    sh "go tool vet #{GO_SOURCE}"
  end

  task :golint do
    sh "golint #{GO_SOURCE}"
  end
end

desc 'Test the code'
task :test do
  sh 'go test -v'
end

desc 'Alias for :install'
task build: :install

desc 'Install for the native platform'
task :install do
  sh 'go install -v'
end

desc 'Build for all supported platforms'
task :xbuild do
  os = %(darwin linux windows)
  arch = 'amd64 ppc64le'
  format = 'build/{{.Dir}}_{{.OS}}_{{.Arch}}'
  sh(
    *%W(
      env CGENABLE=0
      gox -os=#{os} -arch=#{arch} -output=#{format}
    )
  )
end

namespace :build do
  task(:mac64)   { xbuild 'darwin',  'amd64' }
  task(:lin64)   { xbuild 'linux',   'amd64' }
  task(:win64)   { xbuild 'windows', 'amd64' }
  task(:ppc64le) { xbuild 'linux',   'ppc64le' }
end
