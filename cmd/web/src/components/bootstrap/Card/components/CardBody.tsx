import React from 'react';
import {Box} from 'src/components/bootstrap';
import styles from '../Card.module.scss';

export interface Props {
  className?: string;
}

function CardBody({className, ...otherProps}: Props) {
  const scssClasses = [styles['card-body']];

  if (className) {
    scssClasses.push(className);
  }

  return <Box className={scssClasses.join(' ')} {...otherProps} />;
}

export default CardBody;
