GO_SOURCE = Rake::FileList['**/*.go'].tap do |src|
  src.exclude(%r{^vendor/})
  src.exclude(/_test.go$/)
  src.exclude do |f|
    `git ls-files '#{f}'`.empty?
  end
end

GO_TESTS = Rake::FileList['**/*_test.go'].tap do |src|
  src.exclude(%r{^vendor/})
  src.exclude do |f|
    `git ls-files '#{f}'`.empty?
  end
end

GO_PACKAGES = GO_SOURCE.map { |p| "./#{File.dirname(p)}" }.uniq

def run(*array, &block)
  sh(*array.flatten.map(&:to_s), &block)
end

task default: :build

desc 'Clean things up'
task :clean do
  sh 'git clean -xfd'
end

desc 'Fetch all dependencies'
task :setup do
  sh 'go version'
  sh 'go get -u github.com/Masterminds/glide github.com/golang/lint/golint'
  sh 'glide install'
end

desc 'Lint the code'
task lint: %w( lint:vet lint:golint )

namespace :lint do
  task :vet do
    run 'go', 'tool', 'vet', GO_SOURCE, GO_TESTS
  end

  task :golint do
    GO_PACKAGES.each { |pkg| run 'golint', pkg }
  end
end

desc 'Test the code'
task :test do
  run 'go', 'test', '-v', '-cover', GO_PACKAGES
end

desc 'Build for the native platform'
task :build do
  run %w(go install -v)
end

def find_binary(name, os, arch)
  gopath = ENV['GOPATH']
  [
    File.join(gopath, 'bin', "#{os}_#{arch}", name),
    File.join(gopath, 'bin', "#{os}_#{arch}", "#{name}.exe"),
    File.join(gopath, 'bin', name),
    File.join(gopath, 'bin', "#{name}.exe")
  ].select { |b| File.executable? b }.compact.first
end

def xbuild(os, arch)
  mkdir_p 'build'
  sh "env GOOS=#{os} GOARCH=#{arch} go install"
  binary = find_binary('docker-image-cleaner', os, arch)
  mv binary, "build/docker-image-cleaner-#{os}-#{arch}"
end

desc 'Build for all supported platforms'
task xbuild: %w( build:lin64 build:mac64 build:win64 build:ppc64le )

namespace :build do
  task(:mac64)   { xbuild 'darwin',  'amd64' }
  task(:lin64)   { xbuild 'linux',   'amd64' }
  task(:win64)   { xbuild 'windows', 'amd64' }
  task(:ppc64le) { xbuild 'linux',   'ppc64le' }
end
