#!/usr/local/bin/bash
#set -x

# Create functions to backup/restore the cfn template
function backup {
  echo ""
  echo "Backing up Cloudformation template and updating with vars"
  echo ""
  if cp componenttester.yaml componenttester.yaml.bak; then
    echo "Backup successful"
    echo ""
  else
    echo "Backup failed!"
    exit 1
  fi
}

function restore {
  echo ""
  echo "Restoring Cloudformation template to defaults"
  echo ""
  if mv componenttester.yaml.bak componenttester.yaml; then
    echo "Restore successful"
  else
    echo "Restore failed! Check file and do a git --reset"
    exit 1
  fi
}

# Script description, read in the vars
clear
echo "###########################################"
echo "## Welcome to the agogo component tester ##"
echo "###########################################"
echo ""
echo "This script will:"
echo " - ask you for some inputs"
echo " - spin up an ec2 instance"
echo " - run trace/cpuprofile tests"
echo " - zip up the output and upload to s3"
echo " - pull down the zip to your cwd"
echo ""
read -p "Press enter to start"

echo ""
echo ""
echo "Input values for the cfn template, press enter to accept the default"
echo ""
read -p "Branch of gorgonia.org/tensor: " -e -i "v0.9.0-working" TENSORBRANCH
read -p "Branch of gorgonia.org/gorgonia: " -e -i "concurrentTapeMachine" GORGBRANCH
read -p "Component: " -e -i "dualnet" COMPONENT
echo ""
read -p "Instance type: " -e -i "p2.8xlarge" INSTANCETYPE
read -p "Keypair: " -e -i "deploy-ap-southeast-2" KEYPAIR
read -p "Profile name in $HOME/.aws/credentials: " -e -i "agogo" PROFILE
EC2KEY=`grep -A3 $PROFILE $HOME/.aws/credentials | grep aws_access | cut -d' ' -f3`
EC2SECRET=`grep -A3 $PROFILE $HOME/.aws/credentials | grep aws_secret | cut -d' ' -f3`
echo ""
read -p "GitHub User: " -e -i "username" GITUSER
read -p "GitHub Password: " -e -i "password" GITPASS
echo ""

# Backup and do cfn template subs
backup
if sed -i "" -e "s#TENSORBRANCH#$TENSORBRANCH#g" -e "s#GORGBRANCH#$GORGBRANCH#g" -e "s#COMPONENT#$COMPONENT#g" -e "s#INSTANCETYPE#$INSTANCETYPE#g" -e "s#EC2KEY#$EC2KEY#g" -e "s#EC2SECRET#$EC2SECRET#g" -e "s#KEYPAIR#$KEYPAIR#g" -e "s#GITUSER#$GITUSER#g" -e "s#GITPASS#$GITPASS#g" componenttester.yaml; then
  echo "Cloudformation template var substitutions successful"
  echo ""
else
  echo "Cloudformation template var substitutions failed!"
  exit 1
fi

# Create stack, confirm successful
read -p "Enter name for your Cloudformation stack: " -e -i "Test-TraceProfile" STACKNAME
echo ""
echo "Now creating stack..."
echo ""
if aws cloudformation create-stack --stack-name $STACKNAME  --template-body file://componenttester.yaml --profile $PROFILE; then
  echo ""
  echo "Cloudformation request successful"
else
  echo "Cloudformation request failed!"
  restore
  exit 1
fi

# Wait for stack to get created before executing rest of script
echo ""
echo "Now deploying AWS resources & running tests, can take 10-15mins..."
echo ""
if aws cloudformation wait stack-create-complete --stack-name $STACKNAME --profile $PROFILE; then
  echo "Tests complete!"
else
  echo "Tests failed! Examine deploy.out or hit up Gareth on Slack"
fi

# Pull down latest file from S3, could be traces or the output of errors from the instance
echo ""
echo "Pulling down latest file from S3:"
S3OBJECT=`aws s3 ls s3://agogo-testing --recursive --profile $PROFILE | sort | tail -n 1 | awk '{print $4}'`
if aws s3 cp "s3://agogo-testing/$S3OBJECT" ./ --profile $PROFILE; then
  echo "Copy successful"
else
  echo "Copy of $S3OBJECT to cwd failed!"
  restore
  exit 1
fi

# Delete stack
read -p "Press enter to delete stack or 'x' to quit: " DEL
if [ "$DEL" !=  "x" ]; then
  echo ""
  echo "Deleting stack, waiting for DELETE_COMPLETE"
  echo ""
  aws cloudformation delete-stack --stack-name $STACKNAME --profile $PROFILE
  if aws cloudformation wait stack-delete-complete --stack-name $STACKNAME --profile $PROFILE; then
    echo "Stack $STACKNAME deleted."
    echo "All outputs contained in $S3OBJECT"
    restore
    exit 0
  else
    echo "Stack deletion failed! Manually delete $STACKNAME"
    restore
    exit 1
  fi
fi
