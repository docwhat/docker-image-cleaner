
GO_SOURCE = Rake::FileList['**/*.go'].tap do |src|
  src.exclude %r{^vendor/}
  src.exclude do |f|
    `git ls-files '#{f}'`.empty?
  end
end

task default: :build

desc 'Clean things up'
task :clean do
  rm_rf 'vendor'
  rm_rf 'build'
end

desc 'Prepare the build environment'
task :prep do
  sh 'go get -v github.com/Masterminds/glide'
  sh 'glide install'
end

desc 'Test the code'
task :test do
  sh "go tool vet #{GO_SOURCE}"
end

desc 'Build this platform'
task build: %w( prep test ) do
  sh 'go install -v'
end

task travis: %w( prep test build:all )

def build(os, arch)
  mkdir_p 'build'
  sh "env GOOS=#{os} GOARCH=#{arch} go install"
  binary = [
    "#{ENV['GOPATH']}/bin/#{os}_#{arch}/docker-image-cleaner",
    "#{ENV['GOPATH']}/bin/#{os}_#{arch}/docker-image-cleaner.exe",
    "#{ENV['GOPATH']}/bin/docker-image-cleaner",
    "#{ENV['GOPATH']}/bin/docker-image-cleaner.exe"
  ]
    .select { |b| File.executable? b }
    .compact
    .first
  mv binary, "build/docker-image-cleaner-#{os}-#{arch}"
end

namespace :build do
  task all: %w( build:lin64 build:mac64 build:win64 build:ppc64le )
  task(:mac64) { build 'darwin', 'amd64' }
  task(:lin64) { build 'linux', 'amd64' }
  task(:win64) { build 'windows', 'amd64' }
  task(:ppc64le) { build 'linux', 'ppc64le' }
end
