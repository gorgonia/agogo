cpu-image:
		mkdir -p staging/
		cp player_wrapper.sh staging/
	    docker build --no-cache -t "954347443578.dkr.ecr.ap-southeast-2.amazonaws.com/$(CPU_PLAYER_CONTAINER):$(VERSION_TAG)" .
		rm -rf staging/

cpu-push: cpu-image
	    docker push "954347443578.dkr.ecr.ap-southeast-2.amazonaws.com/$(CPU_PLAYER_CONTAINER):$(VERSION_TAG)"
