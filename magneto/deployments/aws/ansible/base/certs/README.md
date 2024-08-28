If you look closely, you'll note that the private keys are missing. I figured
storing them in github would be a bad idea.

I used vault to generate certs, and you can generate new ones if you need them,
with a command that looks something like this:

    this_name="node" ; vault write pki/issue/universal \
        common_name=${this_name}.erixzone.net \
        alt_names="${this_name},${this_name}.staging,${this_name}.staging.erixzone.net,${this_name}.adm,${this_name}.adm.erixzone.net,${this_name}.eng,${this_name}.eng.erixzone.net" \
        ttl=43800h

