go build

dest=~/.terraform.d/plugins/terraform.local/local/lua/0.0.1/linux_amd64/terraform-provider-lua_v0.0.1
mkdir -p $(dirname $dest)

cp terraform-provider-lua $dest 

rm .terraform* -r
~/go/bin/tofu init -reconfigure
~/go/bin/tofu plan
