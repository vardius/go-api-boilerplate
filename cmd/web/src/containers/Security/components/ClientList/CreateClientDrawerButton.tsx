import React, {useRef} from "react";
import {defineMessages, useIntl} from "react-intl";
import {
  Button,
  Center,
  Drawer,
  DrawerBody,
  DrawerCloseButton,
  DrawerContent,
  DrawerOverlay,
  useDisclosure,
} from "@chakra-ui/core";
import CreateClientForm from "./CreateClientForm";

const messages = defineMessages({
  add: {
    id: "create_client.drawer_button.add",
    defaultMessage: "Add",
  },
});

export interface Props {
  onSuccess?: () => void;
}

const CreateClientDrawerButton = (props: Props) => {
  const intl = useIntl();
  const btnRef = useRef(null);
  const {isOpen, onOpen, onClose} = useDisclosure();


  const handleSuccess = async () => {
    if (props.onSuccess) {
      props.onSuccess();
    }

    onClose();
  };

  return (
    <div>
      <Button variant="outline" width="full" colorScheme="green" ref={btnRef} onClick={onOpen}>
        {intl.formatMessage(messages.add)}
      </Button>
      <Drawer
        isOpen={isOpen}
        onClose={onClose}
        finalFocusRef={btnRef}
        size="full"
      >
        <DrawerOverlay/>
        <DrawerContent>
          <DrawerCloseButton border="none"/>
          <DrawerBody>
            <Center minHeight="100vh">
              <CreateClientForm onSuccess={handleSuccess}/>
            </Center>
          </DrawerBody>
        </DrawerContent>
      </Drawer>
    </div>
  );
};

export default CreateClientDrawerButton;
