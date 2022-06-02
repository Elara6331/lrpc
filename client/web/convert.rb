#!/usr/bin/ruby

require 'ruby2js'
require 'ruby2js/filter/functions'

puts Ruby2JS.convert(File.read('lrpc.rb'), eslevel: 2016)