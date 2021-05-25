Pod::Spec.new do |spec|
  spec.name         = 'Gpop'
  spec.version      = '{{.Version}}'
  spec.license      = { :type => 'GNU Lesser General Public License, Version 3.0' }
  spec.homepage     = 'https://github.com/popcateum/go-popcateum'
  spec.authors      = { {{range .Contributors}}
		'{{.Name}}' => '{{.Email}}',{{end}}
	}
  spec.summary      = 'iOS Popcateum Client'
  spec.source       = { :git => 'https://github.com/popcateum/go-popcateum.git', :commit => '{{.Commit}}' }

	spec.platform = :ios
  spec.ios.deployment_target  = '9.0'
	spec.ios.vendored_frameworks = 'Frameworks/Gpop.framework'

	spec.prepare_command = <<-CMD
    curl https://gpopstore.blob.core.windows.net/builds/{{.Archive}}.tar.gz | tar -xvz
    mkdir Frameworks
    mv {{.Archive}}/Gpop.framework Frameworks
    rm -rf {{.Archive}}
  CMD
end
