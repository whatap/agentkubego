package config

func getEc2Hostname() (host string, err error) {

	url := "http://169.254.169.254/latest/meta-data/hostname"
	header := map[string]string{}
	_, imdsV2Token := getImdsv2Token()

	if imdsV2Supported {
		header["X-aws-ec2-metadata-token"] = imdsV2Token
	}
	err = GetHttpWithHeaderResponseLines(url, header, func(respbytes []byte) {
		host = string(respbytes)
	})

	return
}
